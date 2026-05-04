package main

import (
	"fmt"
	"log"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/db"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"github.com/healthcare-market-research/backend/internal/utils/auth"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Connect to database
	if err := db.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Healthcare Market Research - Admin User Seeder ===")
	fmt.Println()

	// Check if admin user already exists
	var count int64
	db.DB.Model(&user.User{}).Where("email = ?", "admin@example.com").Count(&count)

	if count > 0 {
		fmt.Println("✓ Admin user already exists (admin@example.com)")
		fmt.Println("Skipping seed operation.")
		return
	}

	// Create admin user
	fmt.Println("Creating default admin user...")

	hashedPassword, err := auth.HashPassword("Admin@123")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	admin := &user.User{
		Email:        "admin@example.com",
		PasswordHash: hashedPassword,
		Name:         "System Administrator",
		Role:         user.RoleAdmin,
		IsActive:     true,
	}

	if err := db.DB.Create(admin).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Admin user created successfully!")
	fmt.Println()
	fmt.Println("Login Credentials:")
	fmt.Println("------------------")
	fmt.Println("Email:    admin@example.com")
	fmt.Println("Password: Admin@123")
	fmt.Println()
	fmt.Println("⚠️  IMPORTANT: Please change the default password after first login!")
	fmt.Println()
}
