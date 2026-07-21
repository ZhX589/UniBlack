package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/captcha"
	"github.com/ZhX589/UniBlack/backend/internal/config"
	"github.com/ZhX589/UniBlack/backend/internal/db"
	exporter "github.com/ZhX589/UniBlack/backend/internal/export"
	"github.com/ZhX589/UniBlack/backend/internal/handler"
	appMiddleware "github.com/ZhX589/UniBlack/backend/internal/middleware"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/service"
)

func registerPublicEventRoutes(group *echo.Group, getEvent, getCase echo.HandlerFunc) {
	group.GET("/events/:id", getEvent)
	group.GET("/cases/:id", getCase, appMiddleware.CaseDeprecation("/api/v1/events/:id"))
}

func registerLegacyCaseRoutes(group *echo.Group, create, list, get, update, deleteCase, history, evidence, appeals echo.HandlerFunc) *echo.Group {
	legacyCases := group.Group("/cases")
	legacyCases.POST("", create, appMiddleware.CaseDeprecation(""))
	legacyCases.GET("", list, appMiddleware.CaseDeprecation(""))
	legacyCases.GET("/:id", get, appMiddleware.CaseDeprecation("/api/events/:id"))
	legacyCases.PUT("/:id", update, appMiddleware.CaseDeprecation(""))
	legacyCases.DELETE("/:id", deleteCase, appMiddleware.CaseDeprecation(""))
	legacyCases.GET("/:id/history", history, appMiddleware.CaseDeprecation(""))
	legacyCases.GET("/:id/evidence", evidence, appMiddleware.CaseDeprecation(""))
	legacyCases.GET("/:id/appeals", appeals, appMiddleware.CaseDeprecation(""))
	return legacyCases
}

func registerAuthRoutes(group *echo.Group, limiter echo.MiddlewareFunc, register, login, refresh, sendVerificationCode, verifyEmail echo.HandlerFunc) {
	group.Use(limiter)
	group.POST("/register", register)
	group.POST("/login", login)
	group.POST("/refresh", refresh)
	group.POST("/send-verification-code", sendVerificationCode)
	group.POST("/verify-email", verifyEmail)
}

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	// RequestLogger replaces deprecated middleware.Logger (staticcheck SA1019).
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				log.Printf("%s %s %d %s err=%v", v.Method, v.URI, v.Status, v.Latency, v.Error)
			} else {
				log.Printf("%s %s %d %s", v.Method, v.URI, v.Status, v.Latency)
			}
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Connect to database and apply migrations before wiring any handlers.
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.RunMigrations(database); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database connected and migrations applied")

	// Initialize JWT provider
	jwtProvider := auth.NewJWTProvider(auth.JWTConfig{
		Secret:        cfg.JWTSecret,
		RefreshSecret: cfg.RefreshSecret,
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
		Issuer:        "uniblack",
	})

	storageBackend, err := selectStorage(cfg, defaultStorageConstructors())
	if err != nil {
		log.Fatal(err)
	}
	demoCaptcha := captcha.DefaultDemo()

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	subjectRepo := repository.NewSubjectRepository(database)
	caseRepo := repository.NewCaseRepository(database)
	evidenceRepo := repository.NewEvidenceRepository(database)
	submissionRepo := repository.NewSubmissionRepository(database)
	appealRepo := repository.NewAppealRepository(database)
	eventRepo := repository.NewEventRepository(database, storageBackend)
	sanctionRepo := repository.NewSanctionRepository(database)
	auditRepo := repository.NewAuditLogRepository(database)
	settingRepo := repository.NewSystemSettingRepository(database)
	accessListRepo := repository.NewAccessListRepository(database)
	verifyRepo := repository.NewVerificationRepository(database)

	// Initialize services (settings first so OptionMap is ready for auth)
	settingService := service.NewSystemSettingService(settingRepo, accessListRepo, auditRepo)
	if err := settingService.Bootstrap(context.Background()); err != nil {
		log.Fatal(err)
	}
	authService := service.NewAuthService(userRepo, settingService, accessListRepo, verifyRepo, jwtProvider)
	subjectService := service.NewSubjectService(subjectRepo)
	caseService := service.NewCaseService(caseRepo, subjectRepo, auditRepo, storageBackend)
	evidenceService := service.NewEvidenceService(evidenceRepo, caseRepo, storageBackend)
	submissionService := service.NewSubmissionService(submissionRepo, subjectRepo, caseRepo, auditRepo)
	appealService := service.NewAppealService(appealRepo, caseRepo, eventRepo, auditRepo)
	eventService := service.NewEventService(eventRepo, subjectRepo, sanctionRepo, userRepo, authService)
	sanctionService := service.NewSanctionService(sanctionRepo, auditRepo)
	archiveService := exporter.NewArchiveService(subjectRepo, eventRepo, evidenceRepo, storageBackend)
	setupService := service.NewSetupService(repository.NewSetupRepository(database))

	// Seed admin user in dev mode
	if cfg.Environment == "development" || cfg.Environment == "dev" {
		if err := setupService.SeedDevelopmentAdmin(context.Background(), "admin123"); err != nil && !errors.Is(err, service.ErrAlreadyInitialized) {
			log.Printf("Warning: Failed to seed admin: %v", err)
		} else if err == nil {
			settingService.ApplySetupCache("")
		}
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	subjectHandler := handler.NewSubjectHandler(subjectService)
	caseHandler := handler.NewCaseHandler(caseService)
	evidenceHandler := handler.NewEvidenceHandler(evidenceService)
	evidenceHandler.SetEventService(eventService)
	submissionHandler := handler.NewSubmissionHandler(submissionService)
	appealHandler := handler.NewAppealHandler(appealService)
	eventHandler := handler.NewEventHandler(eventService)
	sanctionHandler := handler.NewSanctionHandler(sanctionService)
	archiveHandler := handler.NewArchiveHandler(archiveService)
	settingHandler := handler.NewSystemSettingHandler(settingService)
	userHandler := handler.NewUserManagementHandler(database)
	setupHandler := handler.NewSetupHandler(setupService, settingService)
	verificationHandler := handler.NewVerificationHandler(demoCaptcha)
	publicHandler := handler.NewPublicAPIHandler(subjectService, caseService, evidenceService)

	// Access checks are applied before all public routes. Protected routes add a
	// second check after token authentication so email and username are available.
	publicLimiter := appMiddleware.NewRequestRateLimiter(settingService, "security.rate_limit_public")
	authLimiter := appMiddleware.NewRequestRateLimiter(settingService, "security.rate_limit_auth")
	e.Use(appMiddleware.AccessList(settingService))

	// Public routes
	e.GET("/", func(c echo.Context) error {
		return c.String(200, "UniBlack API Server")
	})

	// Setup routes (public)
	setupGroup := e.Group("/api/setup")
	setupGroup.Use(publicLimiter.Middleware)
	setupGroup.GET("/check", setupHandler.CheckSetup)
	setupGroup.POST("/initialize", setupHandler.Initialize)

	// Auth routes (public)
	authGroup := e.Group("/api/auth")
	registerAuthRoutes(authGroup, authLimiter.Middleware, authHandler.Register, authHandler.Login, authHandler.RefreshToken, authHandler.SendVerificationCode, authHandler.VerifyEmail)
	// Authenticated purpose-scoped codes (submission/appeal) share the same handler.
	e.POST("/api/verification/demo/register", verificationHandler.IssueRegisterDemoToken, publicLimiter.Middleware)

	// Public settings
	settingsPublicGroup := e.Group("/api/settings")
	settingsPublicGroup.Use(publicLimiter.Middleware)
	settingsPublicGroup.GET("/public", settingHandler.GetPublicSettings)

	// Public API routes (no auth required)
	publicGroup := e.Group("/api/v1")
	publicGroup.Use(publicLimiter.Middleware)
	publicGroup.GET("/search", publicHandler.SearchSubjects)
	publicGroup.GET("/lookup", publicHandler.LookupSubject)
	publicGroup.GET("/subjects", publicHandler.ListSubjects)
	publicGroup.GET("/subjects/:id", publicHandler.GetSubject)
	registerPublicEventRoutes(publicGroup, eventHandler.Get, publicHandler.GetCase)
	publicGroup.GET("/subjects/:id/cases", publicHandler.GetCasesBySubject, appMiddleware.CaseDeprecation("/api/v1/subjects/:id"))
	publicGroup.GET("/statistics", publicHandler.GetStatistics)

	// Protected routes
	apiGroup := e.Group("/api")
	apiGroup.Use(appMiddleware.AuthMiddleware(authService))
	apiGroup.Use(appMiddleware.AccessList(settingService))
	apiGroup.Use(authLimiter.Middleware)

	// User routes
	apiGroup.GET("/profile", authHandler.GetProfile)
	apiGroup.POST("/verification/demo/submission", verificationHandler.IssueSubmissionDemoToken)
	apiGroup.POST("/auth/send-verification-code", authHandler.SendVerificationCode)
	apiGroup.GET("/sanctions/me", sanctionHandler.ListMine)
	apiGroup.POST("/sanctions/:id/appeal", sanctionHandler.Appeal)

	// Subject routes (authenticated)
	subjectGroup := apiGroup.Group("/subjects")
	subjectGroup.POST("", subjectHandler.CreateSubject)
	subjectGroup.POST("/publish", eventHandler.Publish)
	subjectGroup.GET("", subjectHandler.ListSubjects)
	subjectGroup.GET("/:id", subjectHandler.GetSubject)
	subjectGroup.PUT("/:id", subjectHandler.UpdateSubject)
	subjectGroup.DELETE("/:id", subjectHandler.DeleteSubject)
	subjectGroup.GET("/:id/cases", caseHandler.GetCasesBySubjectID, appMiddleware.CaseDeprecation("/api/subjects/:id"))

	// Identifier routes (authenticated)
	subjectGroup.POST("/:id/identifiers", subjectHandler.AddIdentifier)
	subjectGroup.GET("/:id/identifiers", subjectHandler.GetIdentifiersBySubjectID)
	subjectGroup.DELETE("/identifiers/:id", subjectHandler.RemoveIdentifier)

	// Case routes (authenticated)
	caseGroup := registerLegacyCaseRoutes(
		apiGroup,
		caseHandler.CreateCase,
		caseHandler.ListCases,
		caseHandler.GetCase,
		caseHandler.UpdateCase,
		caseHandler.DeleteCase,
		caseHandler.GetCaseHistory,
		evidenceHandler.GetEvidenceByCaseID,
		appealHandler.GetAppealsByCaseID,
	)

	// Evidence routes (authenticated)
	evidenceGroup := apiGroup.Group("/evidence")
	evidenceGroup.POST("", evidenceHandler.CreateEvidence)
	evidenceGroup.POST("/upload", evidenceHandler.UploadEvidence)
	evidenceGroup.GET("/:id", evidenceHandler.GetEvidence)
	evidenceGroup.DELETE("/:id", evidenceHandler.DeleteEvidence)

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

	eventGroup := apiGroup.Group("/events")
	eventGroup.GET("/:id", eventHandler.Get)
	eventGroup.GET("/:id/appeals", appealHandler.GetAppealsByEventID)
	eventGroup.POST("/:id/appeals", appealHandler.CreateAppeal)
	eventGroup.POST("/:id/evidence/text", evidenceHandler.CreateEventTextEvidence)
	eventGroup.POST("/:id/evidence/link", evidenceHandler.CreateEventLinkEvidence)
	eventGroup.POST("/:id/evidence/file", evidenceHandler.CreateEventFileEvidence)

	// Review routes (require moderator or admin)
	reviewGroup := caseGroup.Group("/:id/review")
	reviewGroup.Use(appMiddleware.CaseDeprecation(""))
	reviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	reviewGroup.POST("", caseHandler.ReviewCase)

	submissionReviewGroup := submissionGroup.Group("/:id/review")
	submissionReviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	submissionReviewGroup.POST("", submissionHandler.ReviewSubmission)

	appealReviewGroup := appealGroup.Group("/:id/review")
	appealReviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	appealReviewGroup.POST("", appealHandler.ReviewAppeal)
	appealReviewGroup.POST("/resolve", appealHandler.ResolveAppeal)

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
	adminGroup.GET("/sanctions", sanctionHandler.List)
	adminGroup.POST("/sanctions", sanctionHandler.Create)
	adminGroup.POST("/sanctions/:id/revoke", sanctionHandler.Revoke)
	adminGroup.POST("/sanction-appeals/:appealID/resolve", sanctionHandler.ResolveAppeal)
	adminGroup.GET("/exports/subjects/:publicID", archiveHandler.Export)
	adminGroup.POST("/imports/preview", archiveHandler.PreviewImport)
	adminGroup.POST("/imports", archiveHandler.Import)

	// Serve static files (uploads)
	e.Static("/uploads", "./uploads")

	fmt.Printf("Server starting on port %s\n", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
