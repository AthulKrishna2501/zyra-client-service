package repository

import (
	"context"
	"fmt"

	adminModel "github.com/AthulKrishna2501/zyra-admin-service/internals/core/models"
	"github.com/AthulKrishna2501/zyra-auth-service/internals/core/models"
	vendorModel "github.com/AthulKrishna2501/zyra-vendor-service/internals/core/models"
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
