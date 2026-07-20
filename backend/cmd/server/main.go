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

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)

	// Initialize services
	authService := service.NewAuthService(userRepo, jwtProvider)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)

	// Public routes
	e.GET("/", func(c echo.Context) error {
		return c.String(200, "UniBlack API Server")
	})

	// Auth routes
	authGroup := e.Group("/api/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)

	// Protected routes
	apiGroup := e.Group("/api")
	apiGroup.Use(appMiddleware.AuthMiddleware(authService))
	apiGroup.GET("/profile", authHandler.GetProfile)

	// Admin routes (require admin role)
	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(appMiddleware.RequireRole("admin"))
	adminGroup.GET("/dashboard", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"message": "admin dashboard"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s\n", port)
	e.Logger.Fatal(e.Start(":" + port))
}
