package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ZhX589/UniBlack/backend/internal/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

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

	e.GET("/", func(c echo.Context) error {
		return c.String(200, "UniBlack API Server")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s\n", port)
	e.Logger.Fatal(e.Start(":" + port))
}
