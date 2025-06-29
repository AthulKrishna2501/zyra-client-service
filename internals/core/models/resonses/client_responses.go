package resonses

import (
	"time"

	"github.com/google/uuid"
)

type FeaturedVendor struct {
	UserID       uuid.UUID
	FirstName    string
	LastName     string
	Bio          string
	CategoryName string
}

type VendorWithDetails struct {
	UserID          uuid.UUID `json:"vendor_id"`
	UserDetailsName string    `json:"user_details_name"`
}

type VendorWithReview struct {
	ID        string
	UserID    string
	FirstName string
	Rating    float64
	Review    string
}

type ServiceInfo struct {
	ServiceTitle  string
	AvailableDate time.Time
}

type BookingDetails struct {
	ID        uuid.UUID
	BookingID uuid.UUID
	ClientID  uuid.UUID
	VendorID  uuid.UUID
	FirstName string
	LastName  string
	Service   string
	Date      time.Time
	Status    string
	Price     int
	ServiceDuration string
	AdditionalHourPrice int32
	PaymentID string

}
