package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/db"
	"github.com/healthcare-market-research/backend/internal/handler"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/email"
	"github.com/healthcare-market-research/backend/pkg/logger"
	"github.com/healthcare-market-research/backend/pkg/paypal"
	"github.com/healthcare-market-research/backend/pkg/stripe"
	"github.com/joho/godotenv"

	_ "github.com/healthcare-market-research/backend/docs"
)

// @title Healthcare Market Research API
// @version 1.0
// @description RESTful API for healthcare market research reports and categories with caching and pagination support
// @termsOfService https://example.com/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host 192.168.1.2:8081
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Health
// @tag.description Health check endpoints

// @tag.name Authentication
// @tag.description User authentication and token management

// @tag.name Users
// @tag.description User management operations

// @tag.name Reports
// @tag.description Operations related to healthcare market research reports

// @tag.name Categories
// @tag.description Operations related to report categories and hierarchies

// @tag.name Authors
// @tag.description Operations related to report authors and analysts

// @tag.name Audit
// @tag.description Audit log management and tracking

// @tag.name Roles
// @tag.description Role and permission information

// @tag.name Forms
// @tag.description Form submission management (contact forms, sample requests)

// @tag.name Report Images
// @tag.description Internal image management for reports (admin/editor only)

// @tag.name Blogs
// @tag.description Blog post management and publishing

// @tag.name PressReleases
// @tag.description Press release management and publishing
func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Environment)
	logger.Info("Starting Healthcare Market Research API", "environment", cfg.Environment)
	logger.Info("SMTP config loaded", "host", cfg.Email.Host, "port", cfg.Email.Port, "user", cfg.Email.User, "from", cfg.Email.From, "notify_to", cfg.Email.NotifyTo)

	// Connect to database
	if err := db.Connect(cfg); err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis (optional - app continues even if Redis fails)
	if err := cache.Connect(cfg); err != nil {
		logger.Warn("Failed to connect to Redis, caching will be disabled", "error", err)
	} else {
		logger.Info("Redis connected successfully")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	categoryRepo := repository.NewCategoryRepository(db.DB)
	authorRepo := repository.NewAuthorRepository(db.DB)
	reportRepo := repository.NewReportRepository(db.DB, authorRepo)
	auditRepo := repository.NewAuditRepository(db.DB)
	formRepo := repository.NewFormRepository(db.DB)
	reportImageRepo := repository.NewReportImageRepository(db.DB)
	blogRepo := repository.NewBlogRepository(db.DB)
	pressReleaseRepo := repository.NewPressReleaseRepository(db.DB)
	dashboardRepo := repository.NewDashboardRepository(db.DB)
	redirectRepo := repository.NewRedirectRepository(db.DB)
	orderRepo := repository.NewOrderRepository(db.DB)

	// Initialize services
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userRepo, &cfg.Auth)
	cloudflareService := service.NewCloudflareImagesService(&cfg.Cloudflare)
	categoryService := service.NewCategoryService(categoryRepo, cloudflareService)
	reportService := service.NewReportService(reportRepo, reportImageRepo, cloudflareService)
	authorService := service.NewAuthorService(authorRepo, cloudflareService)
	auditService := service.NewAuditService(auditRepo)
	emailService := email.NewSMTPEmailService(&cfg.Email)
	formService := service.NewFormService(formRepo, emailService)
	paypalClient := paypal.NewClient(&cfg.PayPal)
	stripeClient := stripe.NewClient(&cfg.Stripe)
	orderService := service.NewOrderService(orderRepo, reportRepo, paypalClient, stripeClient, emailService)
	reportImageService := service.NewReportImageService(reportImageRepo, reportRepo, cloudflareService)
	blogService := service.NewBlogService(blogRepo)
	pressReleaseService := service.NewPressReleaseService(pressReleaseRepo)
	redirectService := service.NewRedirectService(redirectRepo)
	dashboardService := service.NewDashboardService(
		dashboardRepo, reportRepo, blogRepo, pressReleaseRepo,
		userRepo, formRepo, auditRepo,
	)

	// Initialize scheduler service
	schedulerService := service.NewSchedulerService(reportRepo, blogRepo, pressReleaseRepo)

	// Start scheduler with context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schedulerService.Start(ctx)
	defer schedulerService.Stop()

	// Initialize handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService, auditService)
	userHandler := handler.NewUserHandler(userService, auditService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	reportHandler := handler.NewReportHandler(reportService, authorRepo, auditService)
	authorHandler := handler.NewAuthorHandler(authorService)
	auditHandler := handler.NewAuditHandler(auditService)
	roleHandler := handler.NewRoleHandler()
	formHandler := handler.NewFormHandler(formService)
	reportImageHandler := handler.NewReportImageHandler(reportImageService)
	blogHandler := handler.NewBlogHandler(blogService, auditService)
	pressReleaseHandler := handler.NewPressReleaseHandler(pressReleaseService, auditService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	redirectHandler := handler.NewRedirectHandler(redirectService)
	orderHandler := handler.NewOrderHandler(orderService)
	webhookHandler := handler.NewWebhookHandler(orderService)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Healthcare Market Research API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Global middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Health check endpoint
	app.Get("/health", healthHandler.Check)

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Auth routes (public with rate limiting)
	auth := v1.Group("/auth")
	auth.Post("/login", middleware.RateLimit(cfg.RateLimit.LoginMaxAttempts, cfg.RateLimit.LoginWindow), authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", middleware.RequireAuth(authService), authHandler.Logout)

	// User routes (requires authentication)
	users := v1.Group("/users", middleware.RequireAuth(authService))
	users.Get("/me", userHandler.GetMe)
	users.Get("/", middleware.RequireRole("admin"), userHandler.GetAll)
	users.Get("/by-role/:role", middleware.RequireRole("admin"), userHandler.GetByRole)
	users.Get("/:id", middleware.RequireRole("admin"), userHandler.GetByID)
	users.Post("/", middleware.RequireRole("admin"), userHandler.Create)
	users.Put("/:id", middleware.RequireRole("admin"), userHandler.Update)
	users.Delete("/:id", middleware.RequireRole("admin"), userHandler.Delete)

	// Report routes (public read, protected write)
	v1.Get("/reports", reportHandler.GetAll)
	v1.Get("/reports/link-suggestions", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.GetLinkSuggestions)
	v1.Get("/reports/author/:id", reportHandler.GetByAuthorID)
	v1.Get("/reports/:slug", reportHandler.GetBySlug)
	v1.Get("/search", reportHandler.Search)
	v1.Post("/reports", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.Create)
	v1.Put("/reports/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.Update)
	v1.Patch("/reports/:id/soft-delete", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.SoftDelete)
	v1.Patch("/reports/:id/restore", middleware.RequireAuth(authService), middleware.RequireRole("admin"), reportHandler.Restore)
	v1.Delete("/reports/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), reportHandler.Delete)
	v1.Patch("/reports/:id/schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.SchedulePublish)
	v1.Patch("/reports/:id/cancel-schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.CancelScheduledPublish)

	// Report image routes (admin/editor only)
	v1.Post("/reports/:reportId/images", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportImageHandler.UploadImage)
	v1.Get("/reports/:reportId/images", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportImageHandler.ListImages)
	v1.Get("/reports/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportImageHandler.GetByID)
	v1.Patch("/reports/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportImageHandler.UpdateMetadata)
	v1.Delete("/reports/images/:imageId", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportImageHandler.DeleteImage)

	// Category routes (public read, protected write)
	v1.Get("/categories", categoryHandler.GetAll)
	v1.Get("/categories/:slug", categoryHandler.GetBySlug)
	v1.Get("/categories/:slug/reports", reportHandler.GetByCategorySlug)
	v1.Get("/categories/:slug/blogs", blogHandler.GetByCategorySlug)
	v1.Get("/categories/:slug/press-releases", pressReleaseHandler.GetByCategorySlug)
	v1.Post("/categories/:id/image", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), categoryHandler.UploadImage)

	// Author routes (public read, protected write)
	v1.Get("/authors", authorHandler.GetAll)
	v1.Get("/authors/:id", authorHandler.GetByID)
	v1.Post("/authors", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.Create)
	v1.Put("/authors/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.Update)
	v1.Delete("/authors/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), authorHandler.Delete)
	v1.Post("/authors/:id/image", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.UploadImage)
	v1.Delete("/authors/:id/image", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.DeleteImage)

	// Audit log routes (admin only)
	auditLogs := v1.Group("/audit-logs", middleware.RequireAuth(authService), middleware.RequireRole("admin"))
	auditLogs.Get("/", auditHandler.GetAll)
	auditLogs.Get("/:id", auditHandler.GetByID)

	// Role routes (authenticated users)
	roles := v1.Group("/roles", middleware.RequireAuth(authService))
	roles.Get("/", roleHandler.GetAll)
	roles.Get("/:name", roleHandler.GetByName)

	// Form submission routes (public for create, protected for management)
	forms := v1.Group("/forms")

	// Public endpoint - anyone can submit forms
	forms.Post("/submissions", formHandler.Create)

	// Public read endpoints - no authentication required
	forms.Get("/submissions", formHandler.GetAll)
	forms.Get("/submissions/:id", formHandler.GetByID)
	forms.Get("/submissions/category/:category", formHandler.GetByCategory)
	forms.Get("/stats", formHandler.GetStats)

	// Protected endpoints - require authentication
	forms.Delete("/submissions/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), formHandler.Delete)
	forms.Delete("/submissions", middleware.RequireAuth(authService), middleware.RequireRole("admin"), formHandler.BulkDelete)
	forms.Patch("/submissions/:id/status", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), formHandler.UpdateStatus)

	// Blog routes
	v1.Get("/blogs", blogHandler.GetAll)
	v1.Get("/blogs/slug/:slug", blogHandler.GetBySlug)
	v1.Get("/blogs/:id", blogHandler.GetByID)
	v1.Post("/blogs", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.Create)
	v1.Put("/blogs/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.Update)
	v1.Delete("/blogs/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), blogHandler.Delete)
	v1.Patch("/blogs/:id/submit-review", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.SubmitForReview)
	v1.Patch("/blogs/:id/publish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.Publish)
	v1.Patch("/blogs/:id/unpublish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.Unpublish)
	v1.Patch("/blogs/:id/soft-delete", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.SoftDelete)
	v1.Patch("/blogs/:id/restore", middleware.RequireAuth(authService), middleware.RequireRole("admin"), blogHandler.Restore)
	v1.Patch("/blogs/:id/schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.SchedulePublish)
	v1.Patch("/blogs/:id/cancel-schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), blogHandler.CancelScheduledPublish)

	// Press Release routes
	v1.Get("/press-releases", pressReleaseHandler.GetAll)
	v1.Get("/press-releases/slug/:slug", pressReleaseHandler.GetBySlug)
	v1.Get("/press-releases/:id", pressReleaseHandler.GetByID)
	v1.Post("/press-releases", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.Create)
	v1.Put("/press-releases/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.Update)
	v1.Delete("/press-releases/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), pressReleaseHandler.Delete)
	v1.Patch("/press-releases/:id/submit-review", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.SubmitForReview)
	v1.Patch("/press-releases/:id/publish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.Publish)
	v1.Patch("/press-releases/:id/unpublish", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.Unpublish)
	v1.Patch("/press-releases/:id/soft-delete", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.SoftDelete)
	v1.Patch("/press-releases/:id/restore", middleware.RequireAuth(authService), middleware.RequireRole("admin"), pressReleaseHandler.Restore)
	v1.Patch("/press-releases/:id/schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.SchedulePublish)
	v1.Patch("/press-releases/:id/cancel-schedule", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), pressReleaseHandler.CancelScheduledPublish)

	// Sitemap routes (public, high-limit endpoints for sitemap generation)
	v1.Get("/sitemap/reports", reportHandler.GetSitemap)
	v1.Get("/sitemap/blogs", blogHandler.GetSitemap)
	v1.Get("/sitemap/press-releases", pressReleaseHandler.GetSitemap)

	// Redirect routes
	// Public endpoints (no auth required)
	v1.Get("/redirects/active", redirectHandler.GetActive)
	v1.Post("/redirects/hit/:id", redirectHandler.IncrementHit)

	// Admin endpoints (auth required, admin/editor roles)
	redirectsGroup := v1.Group("/redirects", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"))
	redirectsGroup.Get("/", redirectHandler.GetAll)
	redirectsGroup.Post("/", redirectHandler.Create)
	redirectsGroup.Get("/:id", redirectHandler.GetByID)
	redirectsGroup.Put("/:id", redirectHandler.Update)
	redirectsGroup.Delete("/:id", redirectHandler.Delete)
	redirectsGroup.Patch("/:id/toggle", redirectHandler.Toggle)
	redirectsGroup.Delete("/bulk", middleware.RequireRole("admin"), redirectHandler.BulkDelete)

	// Order routes
	// Stripe webhook — must be registered before body-parser middleware touches it.
	// Fiber v2 makes the raw body available via c.Body() so no special handling needed.
	v1.Post("/webhooks/stripe", webhookHandler.StripeWebhook)

	// Public endpoints — create order + capture payment (no auth required)
	v1.Post("/orders", orderHandler.Create)
	v1.Post("/orders/:id/capture", orderHandler.Capture)
	v1.Post("/orders/:id/stripe-capture", orderHandler.StripeCapture)

	// Protected endpoints — admin/editor only
	ordersGroup := v1.Group("/orders", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"))
	ordersGroup.Get("/", orderHandler.GetAll)
	ordersGroup.Get("/stats", orderHandler.GetStats)
	ordersGroup.Get("/:id", orderHandler.GetByID)
	ordersGroup.Patch("/:id/status", orderHandler.UpdateStatus)

	// Dashboard routes (requires authentication)
	dashboard := v1.Group("/dashboard", middleware.RequireAuth(authService))
	dashboard.Get("/stats", dashboardHandler.GetStats)
	dashboard.Get("/activity", dashboardHandler.GetActivity)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("Server starting", "port", cfg.Port)

	if err := app.Listen(port); err != nil {
		logger.Error("Server failed to start", "error", err)
	}

	// Cleanup
	logger.Info("Running cleanup tasks...")
	if err := db.Close(); err != nil {
		logger.Error("Error closing database", "error", err)
	}
	if err := cache.Close(); err != nil {
		logger.Error("Error closing Redis", "error", err)
	}
	logger.Info("Server shutdown complete")
}
