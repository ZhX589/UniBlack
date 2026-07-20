package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/db"
	"github.com/ZhX589/UniBlack/backend/internal/handler"
	appMiddleware "github.com/ZhX589/UniBlack/backend/internal/middleware"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Rate limiting
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Connect to database
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/uniblack?sslmode=disable"
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
	} else {
		// Run migrations
		if err := db.RunMigrations(database); err != nil {
			log.Printf("Warning: Failed to run migrations: %v", err)
		} else {
			fmt.Println("Database connected and migrations applied")
		}
	}

	// Initialize JWT provider
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-jwt-secret"
	}
	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "dev-refresh-secret"
	}

	jwtProvider := auth.NewJWTProvider(auth.JWTConfig{
		Secret:        jwtSecret,
		RefreshSecret: refreshSecret,
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
		Issuer:        "uniblack",
	})

	// Initialize storage
	storageBackend := storage.NewLocalStorage("./uploads", "http://localhost:8080/uploads")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	subjectRepo := repository.NewSubjectRepository(database)
	caseRepo := repository.NewCaseRepository(database)
	evidenceRepo := repository.NewEvidenceRepository(database)
	submissionRepo := repository.NewSubmissionRepository(database)
	appealRepo := repository.NewAppealRepository(database)
	eventRepo := repository.NewEventRepository(database)
	sanctionRepo := repository.NewSanctionRepository(database)
	auditRepo := repository.NewAuditLogRepository(database)
	settingRepo := repository.NewSystemSettingRepository(database)
	accessListRepo := repository.NewAccessListRepository(database)
	verifyRepo := repository.NewVerificationRepository(database)

	// Initialize services (settings first so OptionMap is ready for auth)
	settingService := service.NewSystemSettingService(settingRepo, accessListRepo, auditRepo)
	if err := settingService.Bootstrap(context.Background()); err != nil {
		log.Printf("Warning: settings bootstrap: %v", err)
	}
	authService := service.NewAuthService(userRepo, settingService, accessListRepo, verifyRepo, jwtProvider)
	subjectService := service.NewSubjectService(subjectRepo)
	caseService := service.NewCaseService(caseRepo, subjectRepo, auditRepo)
	evidenceService := service.NewEvidenceService(evidenceRepo, caseRepo, storageBackend)
	submissionService := service.NewSubmissionService(submissionRepo, subjectRepo, caseRepo, auditRepo)
	appealService := service.NewAppealService(appealRepo, caseRepo, auditRepo)
	eventService := service.NewEventService(eventRepo, sanctionRepo, userRepo, authService)
	sanctionService := service.NewSanctionService(sanctionRepo)

	// Seed admin user in dev mode
	if os.Getenv("GO_ENV") != "production" {
		if err := authService.SeedAdmin(context.Background(), "admin123"); err != nil {
			log.Printf("Warning: Failed to seed admin: %v", err)
		}
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	subjectHandler := handler.NewSubjectHandler(subjectService)
	caseHandler := handler.NewCaseHandler(caseService)
	evidenceHandler := handler.NewEvidenceHandler(evidenceService)
	submissionHandler := handler.NewSubmissionHandler(submissionService)
	appealHandler := handler.NewAppealHandler(appealService)
	eventHandler := handler.NewEventHandler(eventService)
	sanctionHandler := handler.NewSanctionHandler(sanctionService)
	settingHandler := handler.NewSystemSettingHandler(settingService)
	userHandler := handler.NewUserManagementHandler(database)
	setupHandler := handler.NewSetupHandler(authService, settingService)
	publicHandler := handler.NewPublicAPIHandler(subjectService, caseService, evidenceService)

	// Public routes
	e.GET("/", func(c echo.Context) error {
		return c.String(200, "UniBlack API Server")
	})

	// Setup routes (public)
	setupGroup := e.Group("/api/setup")
	setupGroup.GET("/check", setupHandler.CheckSetup)
	setupGroup.POST("/initialize", setupHandler.Initialize)

	// Auth routes (public)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)
	authGroup.POST("/send-verification-code", authHandler.SendVerificationCode)
	authGroup.POST("/verify-email", authHandler.VerifyEmail)

	// Public settings
	settingsPublicGroup := e.Group("/api/settings")
	settingsPublicGroup.GET("/public", settingHandler.GetPublicSettings)

	// Public API routes (no auth required)
	publicGroup := e.Group("/api/v1")
	publicGroup.GET("/search", publicHandler.SearchSubjects)
	publicGroup.GET("/lookup", publicHandler.LookupSubject)
	publicGroup.GET("/subjects", publicHandler.ListSubjects)
	publicGroup.GET("/subjects/:id", publicHandler.GetSubject)
	publicGroup.GET("/subjects/:id/cases", publicHandler.GetCasesBySubject)
	publicGroup.GET("/cases/:id", publicHandler.GetCase)
	publicGroup.GET("/statistics", publicHandler.GetStatistics)

	// Protected routes
	apiGroup := e.Group("/api")
	apiGroup.Use(appMiddleware.AuthMiddleware(authService))

	// User routes
	apiGroup.GET("/profile", authHandler.GetProfile)

	// Subject routes (authenticated)
	subjectGroup := apiGroup.Group("/subjects")
	subjectGroup.POST("", subjectHandler.CreateSubject)
	subjectGroup.POST("/publish", eventHandler.Publish)
	subjectGroup.GET("", subjectHandler.ListSubjects)
	subjectGroup.GET("/:id", subjectHandler.GetSubject)
	subjectGroup.PUT("/:id", subjectHandler.UpdateSubject)
	subjectGroup.DELETE("/:id", subjectHandler.DeleteSubject)
	subjectGroup.GET("/:id/cases", caseHandler.GetCasesBySubjectID)

	// Identifier routes (authenticated)
	subjectGroup.POST("/:id/identifiers", subjectHandler.AddIdentifier)
	subjectGroup.GET("/:id/identifiers", subjectHandler.GetIdentifiersBySubjectID)
	subjectGroup.DELETE("/identifiers/:id", subjectHandler.RemoveIdentifier)

	// Case routes (authenticated)
	caseGroup := apiGroup.Group("/cases")
	caseGroup.POST("", caseHandler.CreateCase)
	caseGroup.GET("", caseHandler.ListCases)
	caseGroup.GET("/:id", caseHandler.GetCase)
	caseGroup.PUT("/:id", caseHandler.UpdateCase)
	caseGroup.DELETE("/:id", caseHandler.DeleteCase)
	caseGroup.GET("/:id/history", caseHandler.GetCaseHistory)

	// Evidence routes (authenticated)
	evidenceGroup := apiGroup.Group("/evidence")
	evidenceGroup.POST("", evidenceHandler.CreateEvidence)
	evidenceGroup.POST("/upload", evidenceHandler.UploadEvidence)
	evidenceGroup.GET("/:id", evidenceHandler.GetEvidence)
	evidenceGroup.DELETE("/:id", evidenceHandler.DeleteEvidence)

	// Case evidence routes
	caseGroup.GET("/:id/evidence", evidenceHandler.GetEvidenceByCaseID)

	// Submission routes (authenticated)
	submissionGroup := apiGroup.Group("/submissions")
	submissionGroup.POST("", submissionHandler.CreateSubmission)
	submissionGroup.GET("", submissionHandler.ListSubmissions)
	submissionGroup.GET("/:id", submissionHandler.GetSubmission)
	submissionGroup.DELETE("/:id", submissionHandler.DeleteSubmission)

	// Appeal routes (authenticated)
	appealGroup := apiGroup.Group("/appeals")
	appealGroup.POST("", appealHandler.CreateAppeal)
	appealGroup.GET("", appealHandler.ListAppeals)
	appealGroup.GET("/:id", appealHandler.GetAppeal)
	appealGroup.DELETE("/:id", appealHandler.DeleteAppeal)

	// Case appeal routes
	caseGroup.GET("/:id/appeals", appealHandler.GetAppealsByCaseID)
	eventGroup := apiGroup.Group("/events")
	eventGroup.GET("/:id", eventHandler.Get)

	// Review routes (require moderator or admin)
	reviewGroup := caseGroup.Group("/:id/review")
	reviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	reviewGroup.POST("", caseHandler.ReviewCase)

	submissionReviewGroup := submissionGroup.Group("/:id/review")
	submissionReviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	submissionReviewGroup.POST("", submissionHandler.ReviewSubmission)

	appealReviewGroup := appealGroup.Group("/:id/review")
	appealReviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	appealReviewGroup.POST("", appealHandler.ReviewAppeal)

	// Admin routes (require admin role)
	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(appMiddleware.RequireRole("admin"))

	// Admin dashboard
	adminGroup.GET("/dashboard", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"message": "admin dashboard"})
	})

	// User management
	adminGroup.GET("/users", userHandler.ListUsers)
	adminGroup.GET("/users/:id", userHandler.GetUser)
	adminGroup.PUT("/users/:id", userHandler.UpdateUser)
	adminGroup.PUT("/users/:id/active", userHandler.ToggleUserActive)
	adminGroup.POST("/users/:id/roles", userHandler.AssignRole)
	adminGroup.DELETE("/users/:id/roles/:role", userHandler.RemoveRole)

	// System settings (NewAPI-style option catalog + CRUD)
	adminGroup.GET("/settings", settingHandler.GetAllSettings)
	adminGroup.GET("/settings/schema", settingHandler.GetSettingsSchema)
	adminGroup.PUT("/settings", settingHandler.UpdateSettings)

	// Access lists
	adminGroup.GET("/access-lists", settingHandler.ListAccessListEntries)
	adminGroup.POST("/access-lists", settingHandler.CreateAccessListEntry)
	adminGroup.DELETE("/access-lists/:id", settingHandler.DeleteAccessListEntry)
	adminGroup.POST("/sanctions", sanctionHandler.Create)
	adminGroup.POST("/sanctions/:id/revoke", sanctionHandler.Revoke)

	// Serve static files (uploads)
	e.Static("/uploads", "./uploads")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s\n", port)
	e.Logger.Fatal(e.Start(":" + port))
}
