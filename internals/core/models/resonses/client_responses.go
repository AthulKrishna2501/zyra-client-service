package resonses

import "github.com/google/uuid"

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
	ID         string
	UserID   string
	FirstName string
	Rating     float64
	Review     string
}
