package models

import (
	"time"

	authModel "github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	"github.com/google/uuid"
)

type Transaction struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID   uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid()"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index"`
	User            authModel.User `gorm:"foreignKey:UserID;references:UserID"`
	PaymentIntentID string         `gorm:"type:varchar(255);index"`
	Purpose         string         `gorm:"not null"`
	AmountPaid      int            `gorm:"not null"`
	PaymentMethod   string         `gorm:"type:varchar(50);not null"`
	PaymentStatus   string         `gorm:"type:varchar(50);not null;index"`
	DateOfPayment   time.Time      `gorm:"not null;index"`
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

type Review struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID  uuid.UUID `gorm:"type:uuid;not null;index"`
	VendorID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Rating    float64   `gorm:"not null;index"`
	Review    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"default:now()"`
	UpdatedAt time.Time `gorm:"default:now()"`

	Client authModel.User `gorm:"foreignKey:ClientID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Vendor authModel.User `gorm:"foreignKey:VendorID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Ticket struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	TicketID  string    `gorm:"type:varchar(255);unique;not null"`
	ClientID  uuid.UUID `gorm:"type:uuid;not null"`
	EventID   uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt time.Time `gorm:"default:current_timestamp"`
	UpdatedAt time.Time `gorm:"default:current_timestamp"`
}

type QR struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null"`
	EventID     uuid.UUID  `gorm:"type:uuid;not null"`
	Code        string     `gorm:"type:text;not null;unique"`
	GeneratedAt time.Time  `gorm:"default:current_timestamp"`
	IsScanned   bool       `gorm:"default:false"`
	ScannedAt   *time.Time `gorm:"type:timestamp"`
}
