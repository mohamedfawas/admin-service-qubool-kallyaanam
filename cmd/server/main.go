// admin-service-qubool-kallyaanam/cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Admin model for GORM
type Admin struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Email    string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
	Role     string `gorm:"not null;default:'admin'"`
	IsActive bool   `gorm:"default:true"`
}

func createDBIfNotExists() error {
	// Connect to default postgres database first
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "postgres"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "admin_db"
	}

	// Check if database exists
	var count int64
	db.Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", dbName).Scan(&count)

	// Create database if it doesn't exist
	if count == 0 {
		log.Printf("Creating database: %s", dbName)
		createSQL := fmt.Sprintf("CREATE DATABASE %s", dbName)
		if err := db.Exec(createSQL).Error; err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Create database if it doesn't exist
	if err := createDBIfNotExists(); err != nil {
		log.Printf("Error creating database: %v", err)
	}

	// Database connection with GORM
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "postgres"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "admin_db"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect using GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
	} else {
		log.Println("Successfully connected to database")

		// Auto migrate schemas
		if err := db.AutoMigrate(&Admin{}); err != nil {
			log.Printf("Error migrating database: %v", err)
		} else {
			log.Println("Database migration successful")
		}
	}

	router := gin.Default()

	// Health Check endpoint
	router.GET("/health", func(c *gin.Context) {
		// Check database health with GORM
		dbStatus := "UP"
		sqlDB, err := db.DB()
		if err != nil {
			dbStatus = "DOWN"
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = "DOWN"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "UP",
			"service":  "admin-service",
			"version":  "0.1.0",
			"database": dbStatus,
		})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":8083",
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests a timeout of 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
