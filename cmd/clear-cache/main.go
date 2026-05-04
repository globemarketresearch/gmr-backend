package main

import (
	"fmt"
	"log"
	"os"

	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Connect to Redis
	if err := cache.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	fmt.Println("Clearing Redis cache for blogs and press releases...")

	// Clear blog caches
	patterns := []string{
		"blog:*",
		"blogs:*",
		"press_release:*",
		"press_releases:*",
	}

	for _, pattern := range patterns {
		fmt.Printf("Clearing pattern: %s\n", pattern)
		if err := cache.DeletePattern(pattern); err != nil {
			log.Printf("Warning: Failed to clear pattern %s: %v", pattern, err)
		}
	}

	fmt.Println("✓ Cache cleared successfully!")
	fmt.Println("\nYou can now restart your backend server.")

	os.Exit(0)
}
