package models

import (
	"time"

	authModel "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/google/uuid"
)

type Transaction struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key"`
	TransactionID uuid.UUID      `json:"transaction_id" gorm:"type:uuid;unique;not null"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Purpose       string         `json:"purpose" gorm:"not null"`
	AmountPaid    int            `json:"amount_paid" gorm:"not null"`
	PaymentMethod string         `json:"payment_method" gorm:"type:varchar(50);not null"`
	PaymentStatus string         `json:"payment_status" gorm:"type:varchar(50);not null;index"`
	DateOfPayment time.Time      `json:"date_of_payment" gorm:"not null;index"`
	User          authModel.User `json:"user" gorm:"foreignKey:UserID"`
}

type Event struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventID   uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"`
	Title     string         `gorm:"type:varchar(255);not null"`
	Location  Location       `json:"location" gorm:"embedded;embeddedPrefix:location_"`
	HostedBy  uuid.UUID      `gorm:"type:uuid;not null;index"`
	User      authModel.User `gorm:"foreignKey:HostedBy;references:UserID"`
	Date      time.Time      `gorm:"type:date;not null;index"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
}

type Location struct {
	Address string  `json:"address"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

type EventDetails struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	EventID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Description    string    `gorm:"type:text"`
	StartTime      time.Time `gorm:"type:time;not null"`
	EndTime        time.Time `gorm:"type:time;not null"`
	PosterImage    string    `gorm:"type:varchar(255)"`
	PricePerTicket int       `gorm:"not null"`
	TicketsSold    int       `gorm:"default:0"`
	TicketLimit    int       `gorm:"not null"`

	Event *Event `gorm:"foreignKey:EventID;references:EventID"`
}
