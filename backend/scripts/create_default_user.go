package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/anttrader?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer pool.Close()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, nickname, role, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO NOTHING`,
		"00000000-0000-0000-0000-000000000001",
		"admin@example.com",
		string(hashedPassword),
		"Admin",
		"admin",
		"active",
	)
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}

	fmt.Println("Default user created successfully!")
	fmt.Println("Email: admin@example.com")
	fmt.Println("Password: admin123")
}
