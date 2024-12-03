package postgres

import (
	"fmt"
	"ghost-approve/internal/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
)

var db *gorm.DB

func InitDB() error {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("SSL_MODE")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s search_path=public",
		dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return err
	}

	return db.AutoMigrate(&models.User{}, &models.Approval{}, &models.ApprovedUser{}, &models.RejectedUser{}, &models.File{}, &models.FileHistory{}, &models.ApprovalReminder{}, &models.HiddenReport{})
}

func GetDB() *gorm.DB {
	return db
}

