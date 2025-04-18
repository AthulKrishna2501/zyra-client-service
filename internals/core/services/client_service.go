package services

import (
	"context"
	"fmt"

	pb "github.com/AthulKrishna2501/proto-repo/client"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"

	authModel "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
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

		err = s.clientRepo.UpdateMasterOfCeremonyStatus(req.GetUserId(), true)
		if err != nil {
			return nil, fmt.Errorf("failed to updated master of ceremony %v", err.Error())
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
				Name:     booking.Vendor.FirstName + "" + booking.Vendor.LastName,
			},
			Service: booking.Service,
			Date:    timestamppb.New(booking.Date),
			Price:   int32(booking.Price),
			Status:  booking.Status,
		})
	}

	return &pb.GetBookingsResponse{
		Bookings: bookingList,
	}, nil
}

func (s *ClientService) BookVendor(ctx context.Context, req *pb.BookVendorRequest) (*pb.BookVendorResponse, error) {
	clientID := req.GetClientId()
	vendorID := req.GetVendorId()
	service := req.GetService()
	date := req.GetDate().AsTime()

	isServiceAvailable, err := s.clientRepo.IsVendorServiceAvailable(ctx, vendorID, service)
	if err != nil {
		s.log.Error("Failed to validate vendor service: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to validate vendor service: %v", err)
	}
	if !isServiceAvailable {
		return nil, status.Errorf(codes.InvalidArgument, "The vendor does not provide the requested service")
	}

	isDateAvailable, err := s.clientRepo.IsVendorAvailableOnDate(ctx, vendorID, date)
	if err != nil {
		s.log.Error("Failed to validate vendor availability: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to validate vendor availability: %v", err)
	}
	if !isDateAvailable {
		return nil, status.Errorf(codes.InvalidArgument, "The vendor is not available on the requested date")
	}

	price, err := s.clientRepo.GetServicePrice(ctx, vendorID, service)
	if err != nil {
		s.log.Error("Failed to fetch service price: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch service price: %v", err)
	}

	booking := &adminModel.Booking{
		ClientID: uuid.MustParse(clientID),
		VendorID: uuid.MustParse(vendorID),
		Service:  service,
		Date:     date,
		Price:    price,
		Status:   "pending",
	}

	err = s.clientRepo.CreateBooking(ctx, booking)
	if err != nil {
		s.log.Error("Failed to create booking: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to create booking: %v", err)
	}

	return &pb.BookVendorResponse{
		Message: "Booking created successfully",
	}, nil
}

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
			PricePerTicket: int32(detail.PricePerTicket),
			TicketsSold:    int32(detail.TicketsSold),
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
		s.log.Error("Failed to fetch upcoming events: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch upcoming events: %v", err)
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
			TicketsSold:    int32(detail.TicketsSold),
			TicketLimit:    int32(detail.TicketLimit),
		})
	}

	return &pb.GetUpcomingEventsResponse{
		Events: eventList,
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

	var categoryList []*pb.Category
	for _, category := range categories {
		categoryList = append(categoryList, &pb.Category{
			CategoryId:   category.ID.String(),
			CategoryName: category.CategoryName,
		})
	}

	vendorDetails := &pb.VendorDetails{
		VendorId:     vendor.UserID.String(),
		FirstName:    vendor.FirstName,
		LastName:     vendor.LastName,
		Categories:   categoryList,
		ProfileImage: vendor.ProfileImage,
	}

	return &pb.GetVendorProfileResponse{
		VendorDetails: vendorDetails,
	}, nil
}
