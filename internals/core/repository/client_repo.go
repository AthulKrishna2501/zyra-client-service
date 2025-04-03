package repository

import (
	"fmt"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
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
		fmt.Println("Error updating Master of Ceremony:", result.Error)
		return fmt.Errorf("failed to update Master of Ceremony status: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		fmt.Println("No client found with ID:", clientID)
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
