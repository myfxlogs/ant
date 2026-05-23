package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID           string `gorm:"type:uuid;primary_key"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Nickname     string
	Role         string `gorm:"not null;default:'user'"`
	Status       string `gorm:"not null;default:'active'"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=anttrader port=5432"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	user := User{
		ID:           "00000000-0000-0000-0000-000000000001",
		Email:        "admin@example.com",
		PasswordHash: string(hashedPassword),
		Nickname:     "Admin",
		Role:         "admin",
		Status:       "active",
	}

	if err := db.Create(&user).Error; err != nil {
		log.Fatal("Failed to create user:", err)
	}

	fmt.Println("Default user created successfully!")
	fmt.Println("Email: admin@example.com")
	fmt.Println("Password: admin123")
}
