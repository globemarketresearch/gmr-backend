package db

import (
	"fmt"
	"log"
	"time"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/domain/audit"
	"github.com/healthcare-market-research/backend/internal/domain/author"
	"github.com/healthcare-market-research/backend/internal/domain/blog"
	"github.com/healthcare-market-research/backend/internal/domain/category"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/domain/order"
	"github.com/healthcare-market-research/backend/internal/domain/press_release"
	"github.com/healthcare-market-research/backend/internal/domain/redirect"
	"github.com/healthcare-market-research/backend/internal/domain/report"
	"github.com/healthcare-market-research/backend/internal/domain/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	var dsn string

	// Prefer DATABASE_URL if available (Railway, Heroku, etc.)
	if cfg.Database.URL != "" {
		dsn = cfg.Database.URL
	} else {
		// Build DSN from individual components
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
			cfg.Database.SSLMode,
		)
	}

	logLevel := logger.Silent
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("Database connected successfully")
	return nil
}

func Migrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Drop sub_categories and market_segments tables if they exist
	DB.Exec("DROP TABLE IF EXISTS market_segments CASCADE")
	DB.Exec("DROP TABLE IF EXISTS sub_categories CASCADE")

	// Run AutoMigrate to create/update tables
	err := DB.AutoMigrate(
		&user.User{},
		&category.Category{},
		&author.Author{},
		&report.Report{},
		&report.ChartMetadata{},
		&report.ReportVersion{},
		&report.ReportImage{},
		&audit.AuditLog{},
		&form.FormSubmission{},
		&blog.Blog{},
		&press_release.PressRelease{},
		&redirect.Redirect{},
		&order.Order{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Clean up old columns if they exist (after tables are created)
	DB.Exec("ALTER TABLE IF EXISTS reports DROP COLUMN IF EXISTS sub_category_id")
	DB.Exec("ALTER TABLE IF EXISTS reports DROP COLUMN IF EXISTS market_segment_id")

	// Add foreign key constraint for report_images.uploaded_by (GORM doesn't auto-create this)
	// Check if constraint already exists before adding
	var constraintExists bool
	DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints
			WHERE constraint_name = 'fk_report_images_user'
			AND table_name = 'report_images'
			AND table_schema = CURRENT_SCHEMA()
		)
	`).Scan(&constraintExists)

	if !constraintExists {
		DB.Exec(`
			ALTER TABLE report_images
			ADD CONSTRAINT fk_report_images_user
			FOREIGN KEY (uploaded_by)
			REFERENCES users(id)
			ON DELETE SET NULL
		`)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
