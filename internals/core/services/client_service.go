package services

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/AthulKrishna2501/proto-repo/client"
	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/cloudinary"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-client-service/internals/logger"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClientService struct {
	pb.UnimplementedClientServiceServer
	clientRepo repository.ClientRepository
	config     config.Config
	log        logger.Logger
}

func NewClientService(clientRepo repository.ClientRepository, cfg config.Config, logger logger.Logger) *ClientService {
	return &ClientService{clientRepo: clientRepo, config: cfg, log: logger}
}

func (s *ClientService) GetMasterOfCeremony(ctx context.Context, req *pb.MasterOfCeremonyRequest) (*pb.MasterOfCeremonyResponse, error) {
	Amount := 250000

	sessionParams := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("inr"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Master of Ceremony Service"),
					},
					UnitAmount: stripe.Int64(int64(Amount)),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(s.config.STRIPE_SUCCESS_URL),
		CancelURL:  stripe.String(s.config.STRIPE_CANCEL_URL),
	}

	stripeSession, err := session.New(sessionParams)
	if err != nil {
		return nil, err
	}

	return &pb.MasterOfCeremonyResponse{
		Url: stripeSession.URL,
	}, nil
}

func (s *ClientService) HandleStripeEvent(ctx context.Context, req *pb.StripeWebhookRequest) (*pb.StripeWebhookResponse, error) {
	s.log.Info("Received Stripe Event: %s", req.EventType)
	s.log.Info("Raw Payload: %s", req.Payload)
	var event stripe.Event
	err := json.Unmarshal([]byte(req.Payload), &event)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %v", err)
	}

	switch req.EventType {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return nil, fmt.Errorf("failed to parse session: %v", err)
		}

		clientID, ok := session.Metadata["client_id"]
		if !ok {
			return nil, fmt.Errorf("client ID not found in metadata")
		}

		s.log.Info("Payment successful! Session ID: %s\n", session.ID)
		err = s.clientRepo.UpdateMasterOfCeremonyStatus(clientID, true)
		if err != nil {
			return nil, fmt.Errorf("failed to update Master of Ceremony status: %v", err)
		}
	default:
		s.log.Info("Received unknown event: %s\n", req.EventType)
	}

	return &pb.StripeWebhookResponse{Status: "Success"}, nil
}

func (s *ClientService) VerifyPayment(ctx context.Context, req *pb.VerifyPaymentRequest) (*pb.VerifyPaymentResponse, error) {
	const amount = 2500
	stripeSession, err := session.Get(req.SessionId, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %v", err)
	}

	if stripeSession.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		err := s.clientRepo.CreditAdminWallet(amount, s.config.ADMIN_EMAIL)
		if err != nil {
			return nil, fmt.Errorf("failed to update createAdminWallet %v", err.Error())
		}
		return &pb.VerifyPaymentResponse{
			Success: true,
			Message: "Payment successful",
		}, nil
	} else {
		return &pb.VerifyPaymentResponse{
			Success: true,
			Message: "The payment was not completed or was canceled.",
		}, nil
	}
}

func (s *ClientService) ClientDashboard(ctx context.Context, req *pb.LandingPageRequest) (*pb.LandingPageResponse, error) {
	categories, err := s.clientRepo.GetCategories(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get Category")
	}

	var categoryList []*pb.Category
	for _, cat := range categories {
		categoryList = append(categoryList, &pb.Category{
			CategoryId:   cat.CategoryID.String(),
			CategoryName: cat.CategoryName,
		})
	}

	return &pb.LandingPageResponse{
		Data: &pb.LandingPageData{
			Categories: categoryList,
		},
	}, nil

}

func (s *ClientService) CreateEvent(ctx context.Context, req *pb.CreateEventRequest) (*pb.CreateEventResponse, error) {
	s.log.Info("UserID in Client Service :", req.GetHostedBy())
	isMasterOfCeremony, err := s.clientRepo.IsMaterofCeremony(ctx, req.GetHostedBy())

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find isMasterofCeremony")
	}

	if !isMasterOfCeremony {
		return nil, status.Error(codes.Unauthenticated, "The user is not a master of ceremony")
	}

	eventDate := req.GetDate().AsTime()
	startTime := req.GetEventDetails().GetStartTime().AsTime()
	endTime := req.GetEventDetails().GetEndTime().AsTime()

	HostedByUUID, _ := uuid.Parse(req.GetHostedBy())
	EventUUID, _ := uuid.Parse(req.GetEventId())

	event := models.Event{
		EventID:  EventUUID,
		Title:    req.GetTitle(),
		Date:     eventDate,
		HostedBy: HostedByUUID,
		Location: models.Location{
			Address: req.GetLocation().GetAddress(),
			City:    req.GetLocation().GetCity(),
			Country: req.GetLocation().GetCountry(),
			Lat:     req.GetLocation().GetLatitude(),
			Lng:     req.GetLocation().GetLongitude(),
		},
	}

	posterImage := req.GetEventDetails().GetPosterImage()

	url, result, err := cloudinary.UploadImage(posterImage)
	if err != nil {
		s.log.Error("failed to upload image to cloudinary %v", err)
		return nil, status.Errorf(codes.Internal, "failed to upload image to cloudinary %v", err)
	}

	s.log.Info("Image Url in cloudinary :", url)
	s.log.Info("Result in Upload Image resp :", result)
	EventDetails := &models.EventDetails{
		EventID:        EventUUID,
		Description:    req.GetEventDetails().GetDescription(),
		StartTime:      startTime,
		EndTime:        endTime,
		PosterImage:    url,
		PricePerTicket: int(req.GetEventDetails().GetPricePerTicket()),
		TicketLimit:    int(req.GetEventDetails().GetTicketLimit()),
	}

	if err := s.clientRepo.CreateEvent(ctx, &event); err != nil {
		s.log.Error("Error creating event: %v", err)

		return nil, status.Errorf(codes.Internal, "failed to create event %v", err)
	}

	if err := s.clientRepo.CreateLocation(ctx, &event.Location); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create location %v", err)
	}

	if err := s.clientRepo.CreateEventDetails(ctx, EventDetails); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event details %v", err)
	}

	return &pb.CreateEventResponse{
		Message: "Event Created Successfully",
	}, nil

}

func (s *ClientService) EditEvent(ctx context.Context, req *pb.EditEventRequest) (*pb.EditEventResponse, error) {
	s.log.Info("Editing Event with ID :", req.GetEventId())

	EventUUID, err := uuid.Parse(req.GetEventId())

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Event ID %v", err)
	}

	HostedUUID, err := uuid.Parse(req.GetHostedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid HostedBy ID %v", err)
	}

	event := models.Event{
		EventID:  EventUUID,
		Title:    req.GetTitle(),
		Date:     req.GetDate().AsTime(),
		HostedBy: HostedUUID,
		Location: models.Location{
			Address: req.GetLocation().Address,
			City:    req.GetLocation().City,
			Country: req.GetLocation().Country,
			Lat:     req.GetLocation().Latitude,
			Lng:     req.GetLocation().Longitude,
		},
	}

	EventDetails := &models.EventDetails{
		EventID:        EventUUID,
		Description:    req.GetEventDetails().GetDescription(),
		StartTime:      req.GetEventDetails().GetStartTime().AsTime(),
		EndTime:        req.GetEventDetails().EndTime.AsTime(),
		PricePerTicket: int(req.GetEventDetails().GetPricePerTicket()),
		TicketLimit:    int(req.GetEventDetails().GetTicketLimit()),
	}

	if err := s.clientRepo.UpdateEvent(ctx, &event); err != nil {
		s.log.Error("Error updating event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to update event: %v", err)
	}

	// if err := s.clientRepo.UpdateLocation(ctx, &event.Location); err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to update location: %v", err)
	// }

	if err := s.clientRepo.UpdateEventDetails(ctx, EventDetails); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update event details: %v", err)
	}

	return &pb.EditEventResponse{
		Message: "Event updated successfully",
	}, nil

}
