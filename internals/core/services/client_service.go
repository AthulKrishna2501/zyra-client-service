package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/AthulKrishna2501/proto-repo/client"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"

	authModel "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/cloudinary"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/repository"
	"github.com/AthulKrishna2501/zyra-client-service/internals/logger"
	"github.com/AthulKrishna2501/zyra-client-service/internals/utils"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/timestamppb"
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

func (s *ClientService) CreateBookingSession(ctx context.Context, req *pb.GenericBookingRequest) (*pb.GenericBookingResponse, error) {

	s.log.Info("UserID in GetService:", req.UserId)

	if req.ServiceType == "master_of_ceremony" {
		Amount := 2500 * 100

		isMasterOfCeremony, err := s.clientRepo.IsMaterofCeremony(ctx, req.UserId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check isMasterOfCeremony %v", err)
		}
		if isMasterOfCeremony {
			return &pb.GenericBookingResponse{
				Message: "The user is already designated as Master of Ceremony",
			}, nil
		}
		sessionParams := &stripe.CheckoutSessionParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String("inr"),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("Master Of Ceremony"),
						},
						UnitAmount: stripe.Int64(int64(Amount)),
					},
					Quantity: stripe.Int64(1),
				},
			},
			Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
			SuccessURL:        stripe.String(fmt.Sprintf("%s&purpose=%s", s.config.STRIPE_SUCCESS_URL, "master_of_ceremony")),
			CancelURL:         stripe.String(s.config.STRIPE_CANCEL_URL),
			ClientReferenceID: stripe.String(req.GetUserId()),
			PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
				Metadata: map[string]string{
					"user_id": req.GetUserId(),
				},
			},
		}

		stripeSession, err := session.New(sessionParams)
		if err != nil {
			return nil, err
		}

		return &pb.GenericBookingResponse{
			Url: stripeSession.URL,
		}, nil

	}

	if req.ServiceType == "vendor_booking" {
		if req.Metadata["vendor_id"] == "" {
			return nil, status.Errorf(codes.InvalidArgument, "vendor_id is required for vendor booking")
		}

		if req.Metadata["service_id"] == "" {
			return nil, status.Errorf(codes.InvalidArgument, "service_id is required for vendor booking")
		}

		count, err := s.clientRepo.GetBookingCount(ctx, req.GetUserId())

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check booking count :%v", err)
		}

		const MaxBookingsPerDay = 3

		if count >= MaxBookingsPerDay {
			return nil, status.Errorf(codes.PermissionDenied, "Booking limit reached for today")
		}

		vendorExists, err := s.clientRepo.VendorExists(ctx, req.Metadata["vendor_id"])

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check vendor exists :%v", err)
		}

		if !vendorExists {
			return nil, status.Errorf(codes.NotFound, "vendor with ID %s does not exists ", req.Metadata["vendor_id"])
		}

		serviceExists, err := s.clientRepo.ServiceExists(ctx, req.Metadata["vendor_id"], req.Metadata["service_id"])

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check service exists :%v", err)
		}

		if !serviceExists {
			return nil, status.Errorf(codes.NotFound, "service with ID %s does not exists", req.Metadata["service_id"])
		}

		ServicePrice, err := s.clientRepo.GetServiceAmount(ctx, req.Metadata["service_id"])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get service price: %v", err)
		}

		totalPrice := ServicePrice * 100

		sessionParams := &stripe.CheckoutSessionParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String("inr"),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("Service Booking"),
						},
						UnitAmount: stripe.Int64(int64(totalPrice)),
					},
					Quantity: stripe.Int64(1),
				},
			},
			Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
			SuccessURL:        stripe.String(fmt.Sprintf("%s&purpose=%s", s.config.STRIPE_SUCCESS_URL, "vendor_booking")),
			CancelURL:         stripe.String(s.config.STRIPE_CANCEL_URL),
			ClientReferenceID: stripe.String(req.GetUserId()),
			Metadata: map[string]string{
				"user_id":    req.GetUserId(),
				"vendor_id":  req.Metadata["vendor_id"],
				"service_id": req.Metadata["service_id"],
			},
		}
		stripeSession, err := session.New(sessionParams)
		if err != nil {
			return nil, err
		}

		return &pb.GenericBookingResponse{
			Url: stripeSession.URL,
		}, nil
	}

	if req.ServiceType == "event_booking" {
		if req.Metadata["event_id"] == "" {
			return nil, status.Errorf(codes.InvalidArgument, "event is required for event booking")
		}

		bookingExists, err := s.clientRepo.EventExists(ctx, req.Metadata["event_id"])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check event booking exists: %v", err)
		}
		if !bookingExists {
			return nil, status.Errorf(codes.NotFound, "event booking with ID %s does not exist", req.Metadata["booking_id"])
		}

		bookingAmount, err := s.clientRepo.GetEventAmount(ctx, req.Metadata["event_id"])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get event booking amount: %v", err)
		}
		totalAmount := bookingAmount * 100

		sessionParams := &stripe.CheckoutSessionParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency: stripe.String("inr"),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("Event Booking"),
						},
						UnitAmount: stripe.Int64(int64(totalAmount)),
					},
					Quantity: stripe.Int64(1),
				},
			},
			Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
			SuccessURL:        stripe.String(fmt.Sprintf("%s&purpose=%s&event_id=%s", s.config.STRIPE_SUCCESS_URL, "event_booking", req.Metadata["event_id"])),
			CancelURL:         stripe.String(s.config.STRIPE_CANCEL_URL),
			ClientReferenceID: stripe.String(req.GetUserId()),
			Metadata: map[string]string{
				"user_id":  req.GetUserId(),
				"event_id": req.Metadata["event_id"],
			},
		}

		stripeSession, err := session.New(sessionParams)
		if err != nil {
			return nil, err
		}

		return &pb.GenericBookingResponse{
			Url: stripeSession.URL,
		}, nil
	}

	return nil, nil

}

func (s *ClientService) HandleStripeEvent(ctx context.Context, req *pb.StripeWebhookRequest) (*pb.StripeWebhookResponse, error) {
	s.log.Info("Received event type:", req.EventType)
	defaultAmount := 2500

	switch req.EventType {
	case "checkout.session.completed":
		var event stripe.Event
		err := json.Unmarshal([]byte(req.Payload), &event)
		if err != nil {
			return nil, err
		}

		var sessionObj stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sessionObj); err != nil {
			return nil, err
		}

		userIdUUID, _ := uuid.Parse(sessionObj.ClientReferenceID)

		purpose := "Role Upgrade"
		if sessionObj.PaymentIntent == nil {
			return nil, status.Errorf(codes.Internal, "PaymentIntent is nil in session")
		}

		serviceID := sessionObj.Metadata["service_id"]
		vendorID := sessionObj.Metadata["vendor_id"]
		eventID := sessionObj.Metadata["event_id"]

		s.log.Info("ServiceID and Vendor ID in HandleStripeEvent :", serviceID, vendorID)

		vendorUUID, _ := uuid.Parse(vendorID)
		eventUUID, _ := uuid.Parse(eventID)

		if serviceID != "" {
			purpose = "Vendor Booking"
		}

		if eventID != "" {
			purpose = "Event Booking"
		}

		Amount := sessionObj.AmountTotal / 100
		if Amount == 0 {
			Amount = int64(defaultAmount)
		}

		switch purpose {
		case "Role Upgrade":
			newTransaction := &models.Transaction{
				UserID:          userIdUUID,
				Purpose:         purpose,
				AmountPaid:      int(Amount),
				PaymentMethod:   "stripe",
				DateOfPayment:   time.Now(),
				PaymentStatus:   "paid",
				PaymentIntentID: sessionObj.PaymentIntent.ID,
			}

			newAdminWalletTransaction := &adminModel.AdminWalletTransaction{
				Date:   time.Now(),
				Type:   purpose,
				Amount: float64(Amount),
				Status: "succeeded",
			}
			err = s.clientRepo.CreateTransaction(ctx, newTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
			}

			err = s.clientRepo.CreateAdminWalletTransaction(ctx, newAdminWalletTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create admin wallet transaction")
			}

			err = s.clientRepo.CreditAmountToAdminWallet(ctx, float64(Amount), s.config.ADMIN_EMAIL)

			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to credit amount to admin wallet %v", err)
			}

			err = s.clientRepo.MakeMasterOfCeremony(ctx, sessionObj.ClientReferenceID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to make master of ceremony %v", err)

			}

		case "Vendor Booking":
			newTransaction := &models.Transaction{
				UserID:          userIdUUID,
				Purpose:         purpose,
				AmountPaid:      int(Amount),
				PaymentMethod:   "stripe",
				DateOfPayment:   time.Now(),
				PaymentStatus:   "paid",
				PaymentIntentID: sessionObj.PaymentIntent.ID,
			}

			newAdminWalletTransaction := &adminModel.AdminWalletTransaction{
				Date:   time.Now(),
				Type:   purpose,
				Amount: float64(Amount),
				Status: "succeeded",
			}

			serviceInfo, err := s.clientRepo.GetServiceInfo(ctx, serviceID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to fetch service name %v:", err)

			}

			newBooking := &adminModel.Booking{
				ClientID:  userIdUUID,
				VendorID:  vendorUUID,
				Service:   serviceInfo.ServiceTitle,
				Date:      serviceInfo.AvailableDate,
				Status:    "pending",
				Price:     int(Amount),
				CreatedAt: time.Now(),
			}

			err = s.clientRepo.CreateBooking(ctx, newBooking)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to book vendor %v:", err)
			}

			err = s.clientRepo.CreateTransaction(ctx, newTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
			}

			err = s.clientRepo.CreditAmountToAdminWallet(ctx, float64(Amount), s.config.ADMIN_EMAIL)

			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to credit amount to admin wallet %v", err)
			}

			err = s.clientRepo.CreateAdminWalletTransaction(ctx, newAdminWalletTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create admin wallet transaction")
			}

		case "Event Booking":
			newTransaction := &models.Transaction{
				UserID:          userIdUUID,
				Purpose:         purpose,
				AmountPaid:      int(Amount),
				PaymentMethod:   "stripe",
				DateOfPayment:   time.Now(),
				PaymentStatus:   "paid",
				PaymentIntentID: sessionObj.PaymentIntent.ID,
			}
			err = s.clientRepo.CreateTransaction(ctx, newTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create event booking transaction: %v", err)
			}

			ticketID := uuid.New()
			qrID := uuid.New()

			newTicket := &models.Ticket{
				ID:        ticketID,
				TicketID:  ticketID.String(),
				ClientID:  userIdUUID,
				EventID:   eventUUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err = s.clientRepo.CreateTicket(ctx, newTicket)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create ticket: %v", err)
			}

			qrCode := utils.GenerateQRCode(ticketID.String())
			newQR := &models.QR{
				ID:          qrID,
				UserID:      userIdUUID,
				EventID:     eventUUID,
				Code:        qrCode,
				GeneratedAt: time.Now(),
				IsScanned:   false,
			}
			err = s.clientRepo.CreateQRCode(ctx, newQR)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create QR code: %v", err)
			}

			newAdminWalletTransaction := &adminModel.AdminWalletTransaction{
				Date:   time.Now(),
				Type:   purpose,
				Amount: float64(Amount),
				Status: "succeeded",
			}
			err = s.clientRepo.CreateAdminWalletTransaction(ctx, newAdminWalletTransaction)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create admin wallet transaction: %v", err)
			}

			err = s.clientRepo.CreditAmountToAdminWallet(ctx, float64(Amount), s.config.ADMIN_EMAIL)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to credit amount to admin wallet: %v", err)
			}

		}

	case "payment_method.attached":
		var paymentMethod stripe.PaymentMethod
		err := json.Unmarshal([]byte(req.Payload), &paymentMethod)
		if err != nil {
			return nil, err
		}

	case "payment_intent.payment_failed":
		var event stripe.Event
		err := json.Unmarshal([]byte(req.Payload), &event)
		if err != nil {
			return nil, err
		}

		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return nil, err
		}

		userIdStr := paymentIntent.Metadata["user_id"]
		userIdUUID, _ := uuid.Parse(userIdStr)

		newTransaction := &models.Transaction{
			UserID:          userIdUUID,
			Purpose:         "Role Upgrade",
			AmountPaid:      defaultAmount,
			PaymentMethod:   "stripe",
			DateOfPayment:   time.Now(),
			PaymentStatus:   "failed",
			PaymentIntentID: paymentIntent.ID,
		}

		err = s.clientRepo.CreateTransaction(ctx, newTransaction)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
		}

	default:
		s.log.Info("Unhandled event type: %s\n", req.EventType)
	}
	return &pb.StripeWebhookResponse{
		Status: "success",
	}, nil
}

func (s *ClientService) ClientDashboard(ctx context.Context, req *pb.LandingPageRequest) (*pb.LandingPageResponse, error) {
	categories, err := s.clientRepo.GetCategories(ctx)
	if err != nil {

		s.log.Error("Failed to fetch categories: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch categories: %v", err)
	}

	var categoryList []*pb.Category
	for _, cat := range categories {
		categoryList = append(categoryList, &pb.Category{
			CategoryId:   cat.CategoryID.String(),
			CategoryName: cat.CategoryName,
		})
	}

	upcomingEvents, eventDetails, err := s.clientRepo.GetUpcomingEvents(ctx)
	if err != nil {
		s.log.Error("Failed to fetch upcoming events: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch upcoming events: %v", err)
	}

	detailsMap := make(map[uuid.UUID]models.EventDetails)
	for _, d := range eventDetails {
		detailsMap[d.EventID] = d
	}

	var eventList []*pb.Event
	for _, event := range upcomingEvents {
		detail := detailsMap[event.EventID]

		eventList = append(eventList, &pb.Event{
			EventId:     event.EventID.String(),
			Title:       event.Title,
			Date:        event.Date.String(),
			Description: detail.Description,
			Image:       detail.PosterImage,

			Location: &pb.Location{
				Address:   event.Location.Address,
				City:      event.Location.City,
				Country:   event.Location.Country,
				Latitude:  event.Location.Lat,
				Longitude: event.Location.Lng,
			},
		})
	}

	featuredVendors, err := s.clientRepo.GetFeaturedVendors(ctx)
	if err != nil {
		s.log.Error("Failed to fetch featured vendors: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch featured vendors: %v", err)
	}

	var vendorList []*pb.Vendor
	for _, vendor := range featuredVendors {
		vendorList = append(vendorList, &pb.Vendor{
			VendorId: vendor.UserID.String(),
			Name:     vendor.FirstName + " " + vendor.LastName,
			Category: vendor.CategoryName,
		})
	}

	return &pb.LandingPageResponse{
		Success: true,
		Data: &pb.LandingPageData{
			Categories:      categoryList,
			UpcomingEvents:  eventList,
			FeaturedVendors: vendorList,
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
		StartTime:      req.GetEventDetails().GetStartTime().AsTime(),
		EndTime:        req.GetEventDetails().GetEndTime().AsTime(),
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

	if err := s.clientRepo.UpdateEventDetails(ctx, EventDetails); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update event details: %v", err)
	}

	return &pb.EditEventResponse{
		Message: "Event updated successfully",
	}, nil

}

func (s *ClientService) GetClientProfile(ctx context.Context, req *pb.GetClientProfileRequest) (*pb.GetClientProfileResponse, error) {
	clientID := req.GetClientId()

	userDetails, err := s.clientRepo.GetUserDetailsByID(ctx, clientID)
	if err != nil {
		s.log.Error("Failed to fetch client profile: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch client profile: %v", err)
	}

	return &pb.GetClientProfileResponse{
		ClientId:     userDetails.UserID.String(),
		FirstName:    userDetails.FirstName,
		LastName:     userDetails.LastName,
		Email:        userDetails.User.Email,
		ProfileImage: userDetails.ProfileImage,
		PhoneNumber:  userDetails.Phone,
	}, nil
}

func (s *ClientService) EditClientProfile(ctx context.Context, req *pb.EditClientProfileRequest) (*pb.EditClientProfileResponse, error) {
	clientID := req.GetClientId()

	userDetails := &authModel.UserDetails{
		UserID:       uuid.MustParse(clientID),
		FirstName:    req.GetFirstName(),
		LastName:     req.GetLastName(),
		ProfileImage: req.GetProfileImage(),
		Phone:        req.GetPhoneNumber(),
	}

	err := s.clientRepo.UpdateUserDetails(ctx, userDetails)
	if err != nil {
		s.log.Error("Failed to update client profile: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to update client profile: %v", err)
	}

	return &pb.EditClientProfileResponse{
		Message: "Client profile updated successfully",
	}, nil
}

func (s *ClientService) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	clientID := req.GetClientId()
	currentPassword := req.GetCurrentPassword()
	newPassword := req.GetNewPassword()
	confirmPassword := req.GetConfirmPassword()

	if newPassword != confirmPassword {
		return nil, status.Errorf(codes.InvalidArgument, "New password and confirm password do not match")
	}

	user, err := s.clientRepo.GetUserByID(ctx, clientID)
	if err != nil {
		s.log.Error("Failed to fetch user: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch user: %v", err)
	}

	if !s.clientRepo.VerifyPassword(user.Password, currentPassword) {
		return nil, status.Errorf(codes.Unauthenticated, "Current password is incorrect")
	}

	hashedPassword, err := s.clientRepo.HashPassword(newPassword)
	if err != nil {
		s.log.Error("Failed to hash password: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to hash password: %v", err)
	}

	err = s.clientRepo.UpdatePassword(ctx, clientID, hashedPassword)
	if err != nil {
		s.log.Error("Failed to update password: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to update password: %v", err)
	}

	return &pb.ResetPasswordResponse{
		Message: "Password reset successfully",
	}, nil
}

func (s *ClientService) GetBookings(ctx context.Context, req *pb.GetBookingsRequest) (*pb.GetBookingsResponse, error) {
	clientID := req.GetClientId()
	s.log.Info("Client id :", clientID)

	bookings, err := s.clientRepo.GetBookingsByClientID(ctx, clientID)
	if err != nil {
		s.log.Error("Failed to fetch bookings: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch bookings: %v", err)
	}

	var bookingList []*pb.Booking
	for _, booking := range bookings {
		bookingList = append(bookingList, &pb.Booking{
			BookingId: booking.BookingID.String(),
			Vendor: &pb.Vendor{
				VendorId: booking.VendorID.String(),
				Name:     booking.FirstName + " " + booking.LastName,
			},
			Service:             booking.Service,
			Date:                timestamppb.New(booking.Date),
			Price:               int32(booking.Price),
			Status:              booking.Status,
			Duration:            booking.ServiceDuration,
			AdditionalHourPrice: booking.AdditionalHourPrice,
			PaymentId:           booking.PaymentID,
		})
	}

	return &pb.GetBookingsResponse{
		Bookings: bookingList,
	}, nil
}

// func (s *ClientService) BookVendor(ctx context.Context, req *pb.BookVendorRequest) (*pb.BookVendorResponse, error) {
// 	clientID := req.GetClientId()
// 	vendorID := req.GetVendorId()
// 	serviceID := req.GetServiceId()
// 	date := req.GetDate().AsTime()

// 	isServiceAvailable, err := s.clientRepo.IsVendorServiceAvailable(ctx, vendorID, service)
// 	if err != nil {
// 		s.log.Error("Failed to validate vendor service: %v", err)
// 		return nil, status.Errorf(codes.Internal, "Failed to validate vendor service: %v", err)
// 	}
// 	if !isServiceAvailable {
// 		return nil, status.Errorf(codes.InvalidArgument, "The vendor does not provide the requested service")
// 	}

// 	isDateAvailable, err := s.clientRepo.IsVendorAvailableOnDate(ctx, vendorID, date)
// 	if err != nil {
// 		s.log.Error("Failed to validate vendor availability: %v", err)
// 		return nil, status.Errorf(codes.Internal, "Failed to validate vendor availability: %v", err)
// 	}
// 	if !isDateAvailable {
// 		return nil, status.Errorf(codes.InvalidArgument, "The vendor is not available on the requested date")
// 	}

// 	price, err := s.clientRepo.GetServicePrice(ctx, vendorID, serviceID)
// 	if err != nil {
// 		s.log.Error("Failed to fetch service price: %v", err)
// 		return nil, status.Errorf(codes.Internal, "Failed to fetch service price: %v", err)
// 	}

// 	booking := &adminModel.Booking{
// 		ClientID: uuid.MustParse(clientID),
// 		VendorID: uuid.MustParse(vendorID),
// 		Service:  service,
// 		Date:     date,
// 		Price:    price,
// 		Status:   "pending",
// 	}

// 	err = s.clientRepo.CreateBooking(ctx, booking)
// 	if err != nil {
// 		s.log.Error("Failed to create booking: %v", err)
// 		return nil, status.Errorf(codes.Internal, "Failed to create booking: %v", err)
// 	}

// 	return &pb.BookVendorResponse{
// 		Message: "Booking created successfully",
// 	}, nil
// }

func (s *ClientService) GetVendorsByCategory(ctx context.Context, req *pb.GetVendorsByCategoryRequest) (*pb.GetVendorsByCategoryResponse, error) {
	category := req.GetCategory()

	vendors, err := s.clientRepo.GetVendorsByCategory(ctx, category)
	if err != nil {
		s.log.Error("Failed to fetch vendors by category: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch vendors by category: %v", err)
	}

	var vendorList []*pb.VendorWithServices
	for _, vendor := range vendors {
		services, err := s.clientRepo.GetServicesByVendorID(ctx, vendor.UserID)
		if err != nil {
			s.log.Error("Failed to fetch services for vendor: %v", err)
			return nil, status.Errorf(codes.Internal, "Failed to fetch services for vendor: %v", err)
		}

		var serviceList []*pb.Service
		for _, service := range services {
			serviceList = append(serviceList, &pb.Service{
				ServiceId:          service.ID.String(),
				ServiceTitle:       service.ServiceTitle,
				ServiceDescription: service.ServiceDescription,
				ServicePrice:       float64(service.ServicePrice),
			})
		}

		vendorList = append(vendorList, &pb.VendorWithServices{
			VendorId: vendor.UserID.String(),
			Name:     vendor.UserDetailsName,

			Services: serviceList,
		})
	}

	return &pb.GetVendorsByCategoryResponse{
		Vendors: vendorList,
	}, nil
}

func (s *ClientService) GetHostedEvents(ctx context.Context, req *pb.GetHostedEventsRequest) (*pb.GetHostedEventsResponse, error) {
	clientID := req.GetClientId()

	events, details, err := s.clientRepo.GetEventsHostedByClient(ctx, clientID)
	if err != nil {
		s.log.Error("Failed to fetch hosted events: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch hosted events: %v", err)
	}

	detailsMap := make(map[uuid.UUID]models.EventDetails)
	for _, d := range details {
		detailsMap[d.EventID] = d
	}

	var eventList []*pb.HostedEvent
	for _, event := range events {
		detail, ok := detailsMap[event.EventID]
		if !ok {
			continue
		}

		eventList = append(eventList, &pb.HostedEvent{
			EventId: event.EventID.String(),
			Title:   event.Title,
			Location: &pb.Location{
				Address:   event.Location.Address,
				City:      event.Location.City,
				Country:   event.Location.Country,
				Latitude:  event.Location.Lat,
				Longitude: event.Location.Lng,
			},
			Date:           timestamppb.New(event.Date),
			Description:    detail.Description,
			StartTime:      detail.StartTime.Format("15:04"),
			EndTime:        detail.EndTime.Format("15:04"),
			PricePerTicket: int32(detail.PricePerTicket),
			TicketLimit:    int32(detail.TicketLimit),
		})
	}

	return &pb.GetHostedEventsResponse{
		Events: eventList,
	}, nil
}

func (s *ClientService) GetUpcomingEvents(ctx context.Context, req *pb.GetUpcomingEventsRequest) (*pb.GetUpcomingEventsResponse, error) {
	events, details, err := s.clientRepo.GetUpcomingEvents(ctx)
	if err != nil {
		s.log.Error("Failed to fetch upcoming events: %v", err.Error())
		return nil, status.Errorf(codes.Internal, "Failed to fetch upcoming events: %v", err.Error())
	}

	detailsMap := make(map[uuid.UUID]models.EventDetails)
	for _, detail := range details {
		detailsMap[detail.EventID] = detail
	}

	var eventList []*pb.UpcomingEvent
	for _, event := range events {
		detail := detailsMap[event.EventID]

		eventList = append(eventList, &pb.UpcomingEvent{
			EventId: event.EventID.String(),
			Title:   event.Title,
			Location: &pb.Location{
				Address:   event.Location.Address,
				City:      event.Location.City,
				Country:   event.Location.Country,
				Latitude:  event.Location.Lat,
				Longitude: event.Location.Lng,
			},
			Date:           timestamppb.New(event.Date),
			Description:    detail.Description,
			PosterImage:    detail.PosterImage,
			PricePerTicket: int32(detail.PricePerTicket),
			TicketLimit:    int32(detail.TicketLimit),
			StartTime:      timestamppb.New(detail.StartTime),
			EndTime:        timestamppb.New(detail.EndTime),
		})
	}

	return &pb.GetUpcomingEventsResponse{
		Message: "Event Listings",
		Events:  eventList,
	}, nil
}

func (s *ClientService) GetVendorProfile(ctx context.Context, req *pb.GetVendorProfileRequest) (*pb.GetVendorProfileResponse, error) {
	vendorID := req.GetVendorId()

	vendor, err := s.clientRepo.GetVendorDetailsByID(ctx, vendorID)
	if err != nil {
		s.log.Error("Failed to fetch vendor profile: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch vendor profile: %v", err)
	}

	categories, err := s.clientRepo.GetVendorCategories(ctx, vendorID)
	if err != nil {
		s.log.Error("Failed to fetch vendor categories: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch vendor categories: %v", err)
	}

	rating, err := s.clientRepo.GetVendorAverageRating(ctx, vendorID)
	if err != nil {
		s.log.Error("Failed to fetch vendor rating: %v", err.Error())
		rating = 0
		return nil, status.Errorf(codes.Internal, "failed to fetch avg rating %v :", err.Error())
	}

	vendorUUID, _ := uuid.Parse(vendorID)

	services, err := s.clientRepo.GetServicesByVendorID(ctx, vendorUUID)
	if err != nil {
		s.log.Error("Failed to fetch vendor services: %v", err.Error())
		rating = 0
		return nil, status.Errorf(codes.Internal, "failed to fetch vendor services %v :", err.Error())
	}

	var categoryList []*pb.Category
	for _, category := range categories {
		categoryList = append(categoryList, &pb.Category{
			CategoryId:   category.ID.String(),
			CategoryName: category.CategoryName,
		})
	}

	var serviceList []*pb.Service
	for _, service := range services {
		serviceList = append(serviceList, &pb.Service{
			ServiceId:          service.ID.String(),
			ServiceTitle:       service.ServiceTitle,
			ServiceDescription: service.ServiceDescription,
			ServicePrice:       float64(service.ServicePrice),
		})

	}

	vendorDetails := &pb.VendorDetails{
		VendorId:     vendor.UserID.String(),
		FirstName:    vendor.FirstName,
		LastName:     vendor.LastName,
		Categories:   categoryList,
		Services:     serviceList,
		ProfileImage: vendor.ProfileImage,
		Rating:       float32(rating),
	}

	return &pb.GetVendorProfileResponse{
		VendorDetails: vendorDetails,
	}, nil
}

func (s *ClientService) AddReviewRatings(ctx context.Context, req *pb.AddReviewRatingsRequest) (*pb.AddReviewRatingsResponse, error) {
	clientUUID, err := uuid.Parse(req.GetClientId())

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse clientID %v", err)
	}
	vendorUUID, err := uuid.Parse(req.GetVendorId())

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse vendorID %v", err)
	}

	newClientReview := &models.Review{
		ClientID: clientUUID,
		VendorID: vendorUUID,
		Rating:   float64(req.GetRating()),
		Review:   req.GetReview(),
	}

	err = s.clientRepo.AddReviewRatingsOfClient(ctx, newClientReview)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add client review ratings %v ", err)
	}

	return &pb.AddReviewRatingsResponse{
		Message: "Your review and rating have been submitted successfully",
	}, nil

}

func (s *ClientService) EditReviewRatings(ctx context.Context, req *pb.EditReviewRatingsRequest) (*pb.EditReviewRatingsResponse, error) {
	reviewID := req.GetReviewId()
	rating := float64(req.GetRating())
	review := req.GetReview()

	err := s.clientRepo.UpdateReviewRatingsOfClient(ctx, reviewID, review, rating)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update review ratings of client %v", err)
	}

	return &pb.EditReviewRatingsResponse{
		Message: "Your review and rating have been updated successfully",
	}, nil

}

func (s *ClientService) DeleteReviewRatings(ctx context.Context, req *pb.DeleteReviewRequest) (*pb.DeleteReviewResponse, error) {
	reviewID := req.GetReviewId()

	err := s.clientRepo.DeleteReview(ctx, reviewID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete client review: %v", err)
	}

	return &pb.DeleteReviewResponse{
		Message: "The review has been successfully removed",
	}, nil

}

func (s *ClientService) ViewClientReviewRatings(ctx context.Context, req *pb.ViewClientReviewRatingsRequest) (*pb.ViewClientReviewRatingsResponse, error) {
	clientID := req.GetClientId()
	reviewDetails, err := s.clientRepo.GetClientReviewRatings(ctx, clientID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch reviews client: %v", err.Error())
	}

	var pbReviews []*pb.ReviewDetails

	for _, review := range reviewDetails {
		pbReviews = append(pbReviews, &pb.ReviewDetails{
			ReviewId:   review.ID,
			VendorId:   review.UserID,
			VendorName: review.FirstName,
			Rating:     float32(review.Rating),
			Review:     review.Review,
		})
	}

	return &pb.ViewClientReviewRatingsResponse{
		Review: pbReviews,
	}, nil
}

func (s *ClientService) GetWallet(ctx context.Context, req *pb.GetWalletRequest) (*pb.GetWalletResponse, error) {
	clientID := req.GetClientId()
	wallet, err := s.clientRepo.GetClientWallet(ctx, clientID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch client wallet: %v", err)
	}

	return &pb.GetWalletResponse{
		Balance:          float32(wallet.WalletBalance),
		TotalDeposits:    float32(wallet.TotalDeposits),
		TotalWithdrawals: float32(wallet.TotalWithdrawals),
	}, nil

}

func (s *ClientService) GetClientTransactions(ctx context.Context, req *pb.ViewClientTransactionsRequest) (*pb.ViewClientTransactionResponse, error) {
	clientID := req.GetClientId()
	walletTransactions, err := s.clientRepo.GetClientTransactions(ctx, clientID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve admin wallet transactions: %v", err.Error())
	}

	var protoTransactions []*pb.ClientTransaction
	for _, txn := range walletTransactions {
		protoTransactions = append(protoTransactions, &pb.ClientTransaction{
			TransactionId: txn.TransactionID.String(),
			Date:          txn.DateOfPayment.String(),
			Type:          txn.Purpose,
			Amount:        float32(txn.AmountPaid),
			Status:        txn.PaymentStatus,
		})
	}

	return &pb.ViewClientTransactionResponse{
		Transactions: protoTransactions,
	}, nil
}

func (s *ClientService) CompleteServiceBooking(ctx context.Context, req *pb.CompleteServiceBookingRequest) (*pb.CompleteServiceBookingResponse, error) {
	if req.BookingId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "booking_id is required")
	}
	if req.ClientId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client_id is required")
	}
	if req.Status == "" {
		return nil, status.Errorf(codes.InvalidArgument, "status is required")
	}

	booking, err := s.clientRepo.GetBookingById(ctx, req.BookingId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "booking not found: %v", err)
	}

	clientUUID, err := uuid.Parse(req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid client_id format")
	}
	if booking.ClientID != clientUUID {
		return nil, status.Errorf(codes.PermissionDenied, "booking does not belong to the client")
	}

	err = s.clientRepo.UpdateClientApprovalStatus(ctx, req.BookingId, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update client approval status: %v", err)
	}

	if booking.IsVendorApproved && !booking.IsClientApproved {
		booking.IsClientApproved = true
	}

	if booking.IsVendorApproved && booking.IsClientApproved {
		err = s.clientRepo.ReleasePaymentToVendor(ctx, booking.VendorID.String(), float64(booking.Price))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to release payment to vendor: %v", err)
		}

		err = s.clientRepo.MarkBookingAsConfirmedAndReleased(ctx, req.BookingId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mark booking as confirmed: %v", err)
		}
	}

	return &pb.CompleteServiceBookingResponse{
		Message: "Booking completed successfully",
	}, nil
}

func (s *ClientService) CancelVendorBooking(ctx context.Context, req *pb.CancelVendorBookingRequest) (*pb.CancelVendorBookingResponse, error) {

	clientUUID, err := uuid.Parse(req.GetClientId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse clien_id")
	}
	booking, err := s.clientRepo.GetBookingById(ctx, req.BookingId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "booking not found: %v", err)
	}

	if booking.Status == "rejected" || booking.Status == "completed" {
		return &pb.CancelVendorBookingResponse{
			Message: "Booking cannot be canceled as it is already completed or rejected.",
		}, nil
	}

	newTransaction := &models.Transaction{
		UserID:        clientUUID,
		Purpose:       "Cancel Vendor Booking",
		AmountPaid:    booking.Price,
		PaymentMethod: "wallet",
		DateOfPayment: time.Now(),
		PaymentStatus: "refunded",
	}

	newAdminWalletTransaction := &adminModel.AdminWalletTransaction{
		Date:   time.Now(),
		Type:   "Cancel Vendor Booking",
		Amount: float64(booking.Price),
		Status: "withdrawn",
	}

	err = s.clientRepo.CreateTransaction(ctx, newTransaction)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	err = s.clientRepo.CreateAdminWalletTransaction(ctx, newAdminWalletTransaction)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create admin wallet transaction")
	}

	err = s.clientRepo.RefundAmount(ctx, s.config.ADMIN_EMAIL, clientUUID.String(), booking.Price)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to refund amount %v", err)
	}

	err = s.clientRepo.UpdateBookingStatus(ctx, booking.BookingID.String(), "cancelled")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update booking status: %v", err)
	}

	return &pb.CancelVendorBookingResponse{
			Message: "Vendor booking cancelled "},
		nil
}
