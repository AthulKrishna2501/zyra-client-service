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
