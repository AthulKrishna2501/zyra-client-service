package repository

import (
	"context"
	"errors"
	"fmt"

	"time"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	clientModel "github.com/AthulKrishna2501/zyra-client-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/models/resonses"
	vendorModel "github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type ClientStorage struct {
	DB *gorm.DB
}

type ClientRepository interface {
	AddReviewRatingsOfClient(ctx context.Context, newReviewRatings *clientModel.Review) error
	CreateAdminWalletTransaction(ctx context.Context, newAdminWalletTransaction *adminModel.AdminWalletTransaction) error
	CreateBooking(ctx context.Context, booking *adminModel.Booking) error
	CreateEvent(ctx context.Context, event *clientModel.Event) error
	CreateEventDetails(ctx context.Context, eventDetails *clientModel.EventDetails) error
	CreateLocation(ctx context.Context, location *clientModel.Location) error
	CreateTransaction(ctx context.Context, newTransaction *clientModel.Transaction) error
	CreditAdminWallet(amount float64, email string) error
	CreditAmountToAdminWallet(ctx context.Context, amount float64, adminEmail string) error
	DeleteReview(ctx context.Context, reviewID string) error
	GetBookingsByClientID(ctx context.Context, clientID string) ([]resonses.BookingDetails, error)
	GetCategories(ctx context.Context) ([]vendorModel.Category, error)
	GetClientReviewRatings(ctx context.Context, clientID string) ([]*resonses.VendorWithReview, error)
	GetEventsHostedByClient(ctx context.Context, clientID string) ([]clientModel.Event, []clientModel.EventDetails, error)
	GetFeaturedVendors(ctx context.Context) ([]resonses.FeaturedVendor, error)
	GetServiceAmount(ctx context.Context, serviceID string) (float64, error)
	GetServiceInfo(ctx context.Context, serviceID string) (*resonses.ServiceInfo, error)
	GetServicePrice(ctx context.Context, vendorID string, service string) (int, error)
	GetServicesByVendorID(ctx context.Context, vendorID uuid.UUID) ([]vendorModel.Service, error)
	GetUpcomingEvents(ctx context.Context) ([]clientModel.Event, []clientModel.EventDetails, error)
	GetUserByID(ctx context.Context, clientID string) (*models.User, error)
	GetUserDetailsByID(ctx context.Context, clientID string) (*models.UserDetails, error)
	GetVendorAverageRating(ctx context.Context, vendorID string) (float64, error)
	GetVendorCategories(ctx context.Context, vendorID string) ([]vendorModel.Category, error)
	GetVendorDetailsByID(ctx context.Context, vendorID string) (*models.UserDetails, error)
	GetVendorsByCategory(ctx context.Context, category string) ([]resonses.VendorWithDetails, error)
	GetClientWallet(ctx context.Context, clientID string) (*vendorModel.Wallet, error)
	GetClientTransactions(ctx context.Context, clientID string) ([]clientModel.Transaction, error)
	GetBookingById(ctx context.Context, bookingID string) (*adminModel.Booking, error)
	HashPassword(password string) (string, error)
	IsMaterofCeremony(ctx context.Context, clientID string) (bool, error)
	IsVendorAvailableOnDate(ctx context.Context, vendorID string, date time.Time) (bool, error)
	IsVendorServiceAvailable(ctx context.Context, vendorID, service string) (bool, error)
	MakeMasterOfCeremony(ctx context.Context, userID string) error
	ServiceExists(ctx context.Context, vendorID, serviceID string) (bool, error)
	UpdateEvent(ctx context.Context, event *clientModel.Event) error
	UpdateEventDetails(ctx context.Context, details *clientModel.EventDetails) error
	UpdateMasterOfCeremonyStatus(clientID string, status bool) error
	UpdatePassword(ctx context.Context, clientID, hashedPassword string) error
	UpdateReviewRatingsOfClient(ctx context.Context, reviewID, review string, rating float64) error
	UpdateUserDetails(ctx context.Context, userDetails *models.UserDetails) error
	UpdateClientApprovalStatus(ctx context.Context, bookingID string, isApproved bool) error
	UpdateBookingStatus(ctx context.Context, bookingID, status string) error
	VendorExists(ctx context.Context, vendorID string) (bool, error)
	VerifyPassword(hashedPassword, password string) bool
	ReleasePaymentToVendor(ctx context.Context, vendorID string, price float64) error
	MarkBookingAsConfirmedAndReleased(ctx context.Context, bookingID string) error
	EventExists(ctx context.Context, eventID string) (bool, error)
	GetEventAmount(ctx context.Context, eventID string) (float64, error)
	CreateTicket(ctx context.Context, ticket *clientModel.Ticket) error
	CreateQRCode(ctx context.Context, qr *clientModel.QR) error
	RefundAmount(ctx context.Context, adminEmail string, clientID string, amount int) error
	GetBookingCount(ctx context.Context, clientID string) (int, error)
}

func NewClientRepository(db *gorm.DB) ClientRepository {
	return &ClientStorage{
		DB: db,
	}
}
func (r *ClientStorage) UpdateMasterOfCeremonyStatus(clientID string, status bool) error {
	result := r.DB.Model(&models.UserDetails{}).
		Where("user_id = ?", clientID).
		Update("master_of_ceremonies", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update Master of Ceremony status: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no client found with ID %s", clientID)
	}

	fmt.Println("Successfully updated Master of Ceremony status for:", clientID)
	return nil
}

func (r *ClientStorage) CreditAdminWallet(amount float64, email string) error {
	if err := r.DB.Model(&adminModel.AdminWallet{}).
		Where("email = ?", email).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).
		Error; err != nil {
		return status.Errorf(codes.Internal, "Failed to update amount in admin wallet: %v", err.Error())
	}

	return nil

}

func (r *ClientStorage) GetCategories(ctx context.Context) ([]vendorModel.Category, error) {
	var categories []vendorModel.Category
	err := r.DB.WithContext(ctx).
		Select("category_id", "category_name").
		Find(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get categories for ClientDashboard: %w", err)
	}

	return categories, nil
}

func (r *ClientStorage) CreateEvent(ctx context.Context, event *clientModel.Event) error {
	err := r.DB.WithContext(ctx).Create(&event).Error
	if err != nil {
		return err
	}

	return nil

}

func (r *ClientStorage) CreateEventDetails(ctx context.Context, eventDetails *clientModel.EventDetails) error {
	err := r.DB.WithContext(ctx).Create(&eventDetails).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ClientStorage) CreateLocation(ctx context.Context, location *clientModel.Location) error {
	err := r.DB.WithContext(ctx).Create(&location).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ClientStorage) IsMaterofCeremony(ctx context.Context, clientID string) (bool, error) {
	var isMC bool
	if err := r.DB.WithContext(ctx).Select("master_of_ceremonies").Table("user_details").Where("user_id = ?", clientID).Scan(&isMC).Error; err != nil {
		return false, err
	}

	return isMC, nil

}

func (r *ClientStorage) UpdateEvent(ctx context.Context, event *clientModel.Event) error {
	result := r.DB.Model(&event).Where("event_id = ?", event.EventID).Updates(&event)
	return result.Error
}

func (r *ClientStorage) UpdateEventDetails(ctx context.Context, details *clientModel.EventDetails) error {
	result := r.DB.Model(&details).Where("event_id = ?", details.EventID).Updates(&details)
	return result.Error
}

func (r *ClientStorage) GetUserDetailsByID(ctx context.Context, clientID string) (*models.UserDetails, error) {
	var userDetails models.UserDetails
	err := r.DB.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", clientID).
		First(&userDetails).Error
	if err != nil {
		return nil, err
	}
	return &userDetails, nil
}

func (r *ClientStorage) UpdateUserDetails(ctx context.Context, userDetails *models.UserDetails) error {
	err := r.DB.WithContext(ctx).
		Model(&models.UserDetails{}).
		Where("user_id = ?", userDetails.UserID).
		Updates(userDetails).Error
	return err
}

func (r *ClientStorage) GetUserByID(ctx context.Context, clientID string) (*models.User, error) {
	var user models.User
	err := r.DB.WithContext(ctx).Where("user_id = ?", clientID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *ClientStorage) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (r *ClientStorage) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (r *ClientStorage) UpdatePassword(ctx context.Context, clientID, hashedPassword string) error {
	err := r.DB.WithContext(ctx).
		Model(&models.User{}).
		Where("user_id = ?", clientID).
		Update("password", hashedPassword).Error
	return err
}

func (r *ClientStorage) GetUpcomingEvents(ctx context.Context) ([]clientModel.Event, []clientModel.EventDetails, error) {
	var events []clientModel.Event
	err := r.DB.WithContext(ctx).
		Where("date >= ?", time.Now()).
		Find(&events).Error
	if err != nil {
		return nil, nil, err
	}

	var eventIDs []uuid.UUID
	for _, ev := range events {
		eventIDs = append(eventIDs, ev.EventID)
	}

	var details []clientModel.EventDetails
	err = r.DB.WithContext(ctx).
		Where("event_id IN ?", eventIDs).
		Find(&details).Error
	if err != nil {
		return nil, nil, err
	}

	return events, details, nil
}

func (r *ClientStorage) GetFeaturedVendors(ctx context.Context) ([]resonses.FeaturedVendor, error) {
	var vendors []resonses.FeaturedVendor

	err := r.DB.WithContext(ctx).
		Table("user_details").
		Select(`
			user_details.user_id,
			user_details.first_name,
			user_details.last_name,
			AVG(reviews.rating) as rating,
			c.category_name
		`).
		Joins("JOIN users u ON u.user_id = user_details.user_id").
		Joins("JOIN vendor_categories vc ON vc.vendor_id = u.user_id").
		Joins("JOIN categories c ON c.category_id = vc.category_id").
		Joins("LEFT JOIN reviews ON reviews.vendor_id = u.user_id").
		Where("u.role = ?", "vendor").
		Group("user_details.user_id, user_details.first_name, user_details.last_name, c.category_name").
		Order("rating DESC").
		Limit(10).
		Scan(&vendors).Error

	if err != nil {
		return nil, err
	}
	return vendors, nil
}

func (r *ClientStorage) IsVendorServiceAvailable(ctx context.Context, vendorID, service string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&vendorModel.Service{}).
		Where("vendor_id = ? AND service_title = ?", vendorID, service).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ClientStorage) IsVendorAvailableOnDate(ctx context.Context, vendorID string, date time.Time) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&vendorModel.Service{}).
		Where("vendor_id = ? AND available_date = ?", vendorID, date).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *ClientStorage) CreateBooking(ctx context.Context, booking *adminModel.Booking) error {
	err := r.DB.WithContext(ctx).Create(booking).Error
	return err
}

func (r *ClientStorage) GetBookingsByClientID(ctx context.Context, clientID string) ([]resonses.BookingDetails, error) {
	var bookings []resonses.BookingDetails

	err := r.DB.WithContext(ctx).
		Table("bookings").
		Joins("JOIN users ON users.user_id = bookings.vendor_id").
		Joins("JOIN user_details ON user_details.user_id = users.user_id").
		Joins("JOIN services ON services.vendor_id = bookings.vendor_id AND services.service_title = bookings.service").
		Where("bookings.client_id = ?", clientID).
		Select(`
		bookings.booking_id,
		bookings.client_id,
		bookings.service,
		bookings.date,
		bookings.price,
		bookings.status,

		services.service_duration,
		services.additional_hour_price,

		users.user_id AS vendor_id,
		user_details.first_name,
		user_details.last_name
	`).Scan(&bookings).Error

	if err != nil {
		return nil, err
	}
	return bookings, nil
}

func (r *ClientStorage) GetVendorsByCategory(ctx context.Context, category string) ([]resonses.VendorWithDetails, error) {
	var vendors []resonses.VendorWithDetails
	err := r.DB.WithContext(ctx).
		Table("users").
		Joins("JOIN vendor_categories vc ON vc.vendor_id = users.user_id").
		Joins("JOIN categories c ON c.category_id = vc.category_id").
		Joins("JOIN user_details ud ON ud.user_id = users.user_id").
		Where("c.category_name = ?", category).
		Select("users.user_id, ud.first_name AS user_details_name").
		Scan(&vendors).Error
	if err != nil {
		return nil, err
	}
	return vendors, nil
}

func (r *ClientStorage) GetServicesByVendorID(ctx context.Context, vendorID uuid.UUID) ([]vendorModel.Service, error) {
	var services []vendorModel.Service
	err := r.DB.WithContext(ctx).
		Where("vendor_id = ?", vendorID).
		Find(&services).Error
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (r *ClientStorage) GetEventsHostedByClient(ctx context.Context, clientID string) ([]clientModel.Event, []clientModel.EventDetails, error) {
	var events []clientModel.Event
	err := r.DB.WithContext(ctx).
		Where("hosted_by = ?", clientID).
		Find(&events).Error
	if err != nil {
		return nil, nil, err
	}

	var eventIDs []uuid.UUID
	for _, ev := range events {
		eventIDs = append(eventIDs, ev.EventID)
	}

	var details []clientModel.EventDetails
	err = r.DB.WithContext(ctx).
		Where("event_id IN ?", eventIDs).
		Find(&details).Error
	if err != nil {
		return nil, nil, err
	}

	return events, details, nil
}

func (r *ClientStorage) GetVendorDetailsByID(ctx context.Context, vendorID string) (*models.UserDetails, error) {
	var userDetails models.UserDetails
	err := r.DB.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", vendorID).
		First(&userDetails).Error
	if err != nil {
		return nil, err
	}
	return &userDetails, nil
}

func (r *ClientStorage) GetVendorCategories(ctx context.Context, vendorID string) ([]vendorModel.Category, error) {
	var categories []vendorModel.Category
	err := r.DB.WithContext(ctx).
		Joins("JOIN vendor_categories vc ON vc.category_id = categories.category_id").
		Where("vc.vendor_id = ?", vendorID).
		Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *ClientStorage) GetServicePrice(ctx context.Context, vendorID string, service string) (int, error) {
	var price int
	err := r.DB.WithContext(ctx).
		Table("services").
		Where("vendor_id = ? AND service_title = ?", vendorID, service).
		Pluck("service_price", &price).Error

	if err != nil {
		return 0, err
	}
	return price, nil
}

func (r *ClientStorage) CreateTransaction(ctx context.Context, newTransaction *clientModel.Transaction) error {
	return r.DB.WithContext(ctx).Create(newTransaction).Error

}

func (r *ClientStorage) MakeMasterOfCeremony(ctx context.Context, userID string) error {
	return r.DB.Model(&models.UserDetails{}).Where("user_id = ?", userID).Update("master_of_ceremonies", true).Error

}

func (r *ClientStorage) IsUserMasterOfCeremony(ctx context.Context, userID string) (bool, error) {
	var status bool

	err := r.DB.WithContext(ctx).Model(&models.UserDetails{}).Select("master_of_ceremony").Where("user_id = ?", userID).Scan(&status).Error

	if err != nil {
		return false, err
	}

	return status, nil
}

func (r *ClientStorage) CreditAmountToAdminWallet(ctx context.Context, amount float64, adminEmail string) error {
	return r.DB.
		Model(&adminModel.AdminWallet{}).Where("email = ?", adminEmail).
		Updates(map[string]interface{}{
			"balance":        gorm.Expr("balance + ?", amount),
			"total_deposits": gorm.Expr("total_deposits + ?", amount),
		}).Error

}

func (r *ClientStorage) CreateAdminWalletTransaction(ctx context.Context, newAdminWalletTransaction *adminModel.AdminWalletTransaction) error {
	return r.DB.WithContext(ctx).Create(newAdminWalletTransaction).Error
}

func (r *ClientStorage) AddReviewRatingsOfClient(ctx context.Context, newReviewRatings *clientModel.Review) error {
	return r.DB.WithContext(ctx).Create(newReviewRatings).Error
}

func (r *ClientStorage) UpdateReviewRatingsOfClient(ctx context.Context, reviewID, review string, rating float64) error {
	return r.DB.WithContext(ctx).Where("id = ?", reviewID).Updates(&clientModel.Review{Review: review, Rating: rating}).Error

}

func (r *ClientStorage) GetClientReviewRatings(ctx context.Context, clientID string) ([]*resonses.VendorWithReview, error) {
	var review []*resonses.VendorWithReview

	err := r.DB.WithContext(ctx).
		Table("reviews").
		Select("reviews.id,reviews.rating,reviews.review,user_details.user_id, user_details.first_name").
		Joins("JOIN user_details ON reviews.vendor_id = user_details.user_id").
		Where("reviews.client_id = ?", clientID).
		Find(&review).Error

	if err != nil {
		return nil, err

	}
	return review, nil
}

func (r *ClientStorage) DeleteReview(ctx context.Context, reviewID string) error {
	result := r.DB.WithContext(ctx).Where("id = ?", reviewID).Delete(&clientModel.Review{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("no review found with this ID")
	}
	return nil
}

func (r *ClientStorage) GetVendorAverageRating(ctx context.Context, vendorID string) (float64, error) {
	var avg float64
	err := r.DB.WithContext(ctx).
		Table("reviews").
		Select("AVG(rating)").
		Where("vendor_id = ?", vendorID).
		Scan(&avg).Error

	if err != nil {
		return 0, err
	}
	return avg, nil
}

func (r *ClientStorage) VendorExists(ctx context.Context, vendorID string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&models.User{}).Where("user_id = ?", vendorID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check if vendor exists: %v", err)
	}
	return count > 0, nil
}

func (r *ClientStorage) ServiceExists(ctx context.Context, vendorID, serviceID string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&vendorModel.Service{}).Where("vendor_id = ? AND id = ?", vendorID, serviceID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check if service exists: %v", err)
	}
	return count > 0, nil
}

func (r *ClientStorage) GetServiceAmount(ctx context.Context, serviceID string) (float64, error) {
	var ServicePrice float64
	err := r.DB.WithContext(ctx).Model(&vendorModel.Service{}).Select("service_price").Where("id = ?", serviceID).Scan(&ServicePrice).Error
	if err != nil {
		return 0, err
	}

	return ServicePrice, nil
}

func (r *ClientStorage) GetServiceInfo(ctx context.Context, serviceID string) (*resonses.ServiceInfo, error) {
	var serviceInfo resonses.ServiceInfo
	err := r.DB.WithContext(ctx).Model(&vendorModel.Service{}).Select("service_title,available_date").Where("id =?", serviceID).Scan(&serviceInfo).Error
	if err != nil {
		return nil, err
	}

	return &serviceInfo, nil

}

func (r *ClientStorage) GetClientWallet(ctx context.Context, clientID string) (*vendorModel.Wallet, error) {
	var wallet vendorModel.Wallet

	err := r.DB.Where("client_id = ?", clientID).First(&wallet).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		clientIdUUID, err := uuid.Parse(clientID)
		if err != nil {
			return nil, err
		}
		newWallet := &vendorModel.Wallet{
			ClientID:         clientIdUUID,
			WalletBalance:    0,
			TotalDeposits:    0,
			TotalWithdrawals: 0,
		}

		err = r.DB.Create(newWallet).Error
		if err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (r *ClientStorage) GetClientTransactions(ctx context.Context, clientID string) ([]clientModel.Transaction, error) {
	var transactions []clientModel.Transaction

	err := r.DB.WithContext(ctx).Where("user_id = ?", clientID).Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *ClientStorage) GetBookingById(ctx context.Context, bookingID string) (*adminModel.Booking, error) {
	var booking adminModel.Booking
	err := r.DB.WithContext(ctx).Where("booking_id = ?", bookingID).First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *ClientStorage) UpdateClientApprovalStatus(ctx context.Context, bookingID string, isApproved bool) error {
	return r.DB.WithContext(ctx).
		Model(&adminModel.Booking{}).
		Where("booking_id = ?", bookingID).
		Update("is_client_approved", isApproved).Error
}

func (r *ClientStorage) ReleasePaymentToVendor(ctx context.Context, vendorID string, price float64) error {
	var vendorWallet vendorModel.Wallet
	vendorUUID, err := uuid.Parse(vendorID)
	if err != nil {
		return err
	}

	tx := r.DB.WithContext(ctx).Begin()

	err = tx.Where("vendor_id = ?", vendorUUID).First(&vendorWallet).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	vendorWallet.WalletBalance += int64(price)
	vendorWallet.TotalDeposits += int64(price)

	err = tx.Save(&vendorWallet).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	var adminWallet adminModel.AdminWallet
	err = tx.Where("email = ?", "admin@example.com").First(&adminWallet).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if adminWallet.Balance < price {
		tx.Rollback()
		return fmt.Errorf("admin wallet does not have enough balance")
	}

	adminWallet.Balance -= price
	adminWallet.TotalWithdrawals += price

	err = tx.Where("email = ?", "admin@example.com").Save(&adminWallet).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	vendorTransaction := clientModel.Transaction{
		UserID:        vendorUUID,
		Purpose:       "Vendor Booking Payment",
		AmountPaid:    int(price),
		PaymentMethod: "wallet",
		PaymentStatus: "completed",
		DateOfPayment: time.Now(),
	}
	err = tx.Create(&vendorTransaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	adminTransaction := adminModel.AdminWalletTransaction{
		Date:   time.Now(),
		Type:   "Vendor Payment Release",
		Amount: price,
		Status: "withdrawn",
	}
	err = tx.Create(&adminTransaction).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *ClientStorage) MarkBookingAsConfirmedAndReleased(ctx context.Context, bookingID string) error {
	tx := r.DB.WithContext(ctx).Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var booking adminModel.Booking
	if err := tx.Where("booking_id = ?", bookingID).First(&booking).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find booking: %w", err)
	}

	booking.IsVendorApproved = true
	booking.IsClientApproved = true
	booking.IsFundReleased = true
	booking.UpdatedAt = time.Now()

	if err := tx.Save(&booking).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update booking: %w", err)
	}

	return tx.Commit().Error
}

func (r *ClientStorage) EventExists(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&clientModel.Event{}).
		Where("event_id = ?", eventID).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ClientStorage) GetEventAmount(ctx context.Context, eventID string) (float64, error) {
	var event clientModel.EventDetails
	err := r.DB.WithContext(ctx).
		Select("price_per_ticket").
		Where("event_id = ?", eventID).
		First(&event).Error

	if err != nil {
		return 0, err
	}
	return float64(event.PricePerTicket), nil
}

func (r *ClientStorage) CreateTicket(ctx context.Context, ticket *clientModel.Ticket) error {
	if err := r.DB.WithContext(ctx).Create(ticket).Error; err != nil {
		return err
	}
	return nil
}

func (r *ClientStorage) CreateQRCode(ctx context.Context, qr *clientModel.QR) error {
	if err := r.DB.WithContext(ctx).Create(qr).Error; err != nil {
		return err
	}
	return nil
}

func (r *ClientStorage) RefundAmount(ctx context.Context, adminEmail string, clientID string, amount int) error {
	var wallet vendorModel.Wallet
	var adminWallet adminModel.AdminWallet

	err := r.DB.WithContext(ctx).Model(&adminWallet).Where("email = ?", adminEmail).Update("balance", gorm.Expr("balance - ?", amount)).Error
	if err != nil {
		return err
	}

	err = r.DB.WithContext(ctx).Model(&adminWallet).Where("email = ?", adminEmail).Update("total_withdrawals", gorm.Expr("total_withdrawals + ?", amount)).Error

	if err != nil {
		return nil
	}

	err = r.DB.WithContext(ctx).Model(&wallet).Where("client_id = ?", clientID).First(&wallet).Error

	clientUUID, _ := uuid.Parse(clientID)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newWallet := vendorModel.Wallet{
			ClientID:      clientUUID,
			WalletBalance: int64(amount),
		}

		return r.DB.WithContext(ctx).Create(&newWallet).Error
	} else if err != nil {
		return err
	}

	err = r.DB.WithContext(ctx).Model(wallet).Where("client_id = ?", clientID).Update("wallet_balance", gorm.Expr("wallet_balance + ?", amount)).Error

	if err != nil {
		return err
	}

	return r.DB.WithContext(ctx).Model(wallet).Where("client_id = ?", clientID).Update("total_deposits", gorm.Expr("total_deposits + ?", amount)).Error

}

func (r *ClientStorage) UpdateBookingStatus(ctx context.Context, bookingID, status string) error {
	return r.DB.WithContext(ctx).Model(&adminModel.Booking{}).Where("booking_id = ?", bookingID).Update("status", status).Error
}

func (r *ClientStorage) GetBookingCount(ctx context.Context, clientID string) (int, error) {
	var count int64

	err := r.DB.Model(&adminModel.Booking{}).Where("client_id = ? AND DATE(created_at) = CURRENT_DATE", clientID).Count(&count).Error

	if err != nil {
		return 0, err
	}

	return int(count), nil
}
