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
	UpdateMasterOfCeremonyStatus(clientID string, status bool) error
	CreditAdminWallet(amount float64, email string) error
	GetCategories(ctx context.Context) ([]vendorModel.Category, error)
	CreateEvent(ctx context.Context, event *clientModel.Event) error
	CreateEventDetails(ctx context.Context, eventDetails *clientModel.EventDetails) error
	CreateLocation(ctx context.Context, location *clientModel.Location) error
	IsMaterofCeremony(ctx context.Context, clientID string) (bool, error)
	UpdateEvent(ctx context.Context, event *clientModel.Event) error
	UpdateEventDetails(ctx context.Context, details *clientModel.EventDetails) error
	GetUserDetailsByID(ctx context.Context, clientID string) (*models.UserDetails, error)
	UpdateUserDetails(ctx context.Context, userDetails *models.UserDetails) error
	GetUserByID(ctx context.Context, clientID string) (*models.User, error)
	VerifyPassword(hashedPassword, password string) bool
	HashPassword(password string) (string, error)
	UpdatePassword(ctx context.Context, clientID, hashedPassword string) error
	GetBookingsByClientID(ctx context.Context, clientID string) ([]adminModel.Booking, error)
	GetUpcomingEvents(ctx context.Context) ([]clientModel.Event, []clientModel.EventDetails, error)
	GetFeaturedVendors(ctx context.Context) ([]resonses.FeaturedVendor, error)
	IsVendorServiceAvailable(ctx context.Context, vendorID, service string) (bool, error)
	IsVendorAvailableOnDate(ctx context.Context, vendorID string, date time.Time) (bool, error)
	CreateBooking(ctx context.Context, booking *adminModel.Booking) error
	GetVendorsByCategory(ctx context.Context, category string) ([]resonses.VendorWithDetails, error)
	GetServicesByVendorID(ctx context.Context, vendorID uuid.UUID) ([]vendorModel.Service, error)
	GetEventsHostedByClient(ctx context.Context, clientID string) ([]clientModel.Event, []clientModel.EventDetails, error)
	GetVendorDetailsByID(ctx context.Context, vendorID string) (*models.UserDetails, error)
	GetVendorCategories(ctx context.Context, vendorID string) ([]vendorModel.Category, error)
	GetServicePrice(ctx context.Context, vendorID string, service string) (int, error)
	CreateTransaction(ctx context.Context, newTransaction *clientModel.Transaction) error
	MakeMasterOfCeremony(ctx context.Context, userID string) error
	CreditAmountToAdminWallet(ctx context.Context, amount float64, adminEmail string) error
	CreateAdminWalletTransaction(ctx context.Context, newAdminWalletTransaction *adminModel.AdminWalletTransaction) error
	AddReviewRatingsOfClient(ctx context.Context, newReviewRatings *clientModel.Review) error
	UpdateReviewRatingsOfClient(ctx context.Context, reviewID, review string, rating float64) error
	GetClientReviewRatings(ctx context.Context, clientID string) ([]*resonses.VendorWithReview, error)
	DeleteReview(ctx context.Context, reviewID string) error
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
		Joins("JOIN users u ON u.user_id = user_details.user_id").
		Joins("JOIN vendor_categories vc ON vc.vendor_id = u.user_id").
		Joins("JOIN categories c ON c.category_id = vc.category_id").
		Where("u.role = ?", "vendor").
		Select("user_details.user_id, user_details.first_name, user_details.last_name, c.category_name as category_name").
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

func (r *ClientStorage) GetBookingsByClientID(ctx context.Context, clientID string) ([]adminModel.Booking, error) {
	var bookings []adminModel.Booking

	err := r.DB.WithContext(ctx).
		Table("bookings").
		Joins("JOIN users ON users.user_id = bookings.vendor_id").
		Joins("JOIN user_details ON user_details.user_id = users.user_id").
		Where("bookings.client_id = ?", clientID).
		Select(`
		bookings.booking_id,
		bookings.client_id,
		bookings.service,
		bookings.date,
		bookings.price,
		bookings.status,

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
