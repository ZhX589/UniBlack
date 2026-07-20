package main

import (
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
	auditRepo := repository.NewAuditLogRepository(database)

	// Initialize services
	authService := service.NewAuthService(userRepo, jwtProvider)
	subjectService := service.NewSubjectService(subjectRepo)
	caseService := service.NewCaseService(caseRepo, subjectRepo, auditRepo)
	evidenceService := service.NewEvidenceService(evidenceRepo, caseRepo, storageBackend)
	submissionService := service.NewSubmissionService(submissionRepo, subjectRepo, caseRepo, auditRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	subjectHandler := handler.NewSubjectHandler(subjectService)
	caseHandler := handler.NewCaseHandler(caseService)
	evidenceHandler := handler.NewEvidenceHandler(evidenceService)
	submissionHandler := handler.NewSubmissionHandler(submissionService)

	// Public routes
	e.GET("/", func(c echo.Context) error {
		return c.String(200, "UniBlack API Server")
	})

	// Auth routes (public)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)

	// Public search (no auth required)
	e.GET("/api/search", subjectHandler.SearchSubjects)
	e.GET("/api/subjects/lookup", subjectHandler.GetSubjectByIdentifier)

	// Protected routes
	apiGroup := e.Group("/api")
	apiGroup.Use(appMiddleware.AuthMiddleware(authService))

	// User routes
	apiGroup.GET("/profile", authHandler.GetProfile)

	// Subject routes (authenticated)
	subjectGroup := apiGroup.Group("/subjects")
	subjectGroup.POST("", subjectHandler.CreateSubject)
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

	// Review routes (require moderator or admin)
	reviewGroup := caseGroup.Group("/:id/review")
	reviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	reviewGroup.POST("", caseHandler.ReviewCase)

	submissionReviewGroup := submissionGroup.Group("/:id/review")
	submissionReviewGroup.Use(appMiddleware.RequireRole("admin", "moderator"))
	submissionReviewGroup.POST("", submissionHandler.ReviewSubmission)

	// Admin routes (require admin role)
	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(appMiddleware.RequireRole("admin"))
	adminGroup.GET("/dashboard", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"message": "admin dashboard"})
	})

	// Serve static files (uploads)
	e.Static("/uploads", "./uploads")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s\n", port)
	e.Logger.Fatal(e.Start(":" + port))
}
