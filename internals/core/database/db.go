package database

import (
	"log"

	"github.com/AthulKrishna2501/zyra-client-service/internals/app/config"
	"github.com/AthulKrishna2501/zyra-client-service/internals/core/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDatabase(env config.Config) *gorm.DB {
	db, err := gorm.Open(postgres.Open(env.DB_URL), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database", err)
		return nil
	}

	err = AutoMigrate(db)
	if err != nil {
		log.Fatal("Error in automigration", err)
		return nil

	}

	return db
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.Event{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.EventDetails{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Location{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Review{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Transaction{}); err != nil {
		return err
	}
	return nil
}	
