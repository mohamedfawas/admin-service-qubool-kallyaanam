// admin-service-qubool-kallyaanam/cmd/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Database connection
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "postgres" // Default in Docker
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432" // Default PostgreSQL port
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres" // Default user
	}

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres" // Default password
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "admin_db" // Default database name
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Try to connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Error opening database connection: %v", err)
	} else {
		// Test connection
		err = db.Ping()
		if err != nil {
			log.Printf("Error connecting to database: %v", err)
		} else {
			log.Println("Successfully connected to database")
		}
		defer db.Close()
	}

	router := gin.Default()

	// Health Check endpoint
	router.GET("/health", func(c *gin.Context) {
		// Check database health
		dbStatus := "UP"
		if db != nil {
			err := db.Ping()
			if err != nil {
				dbStatus = "DOWN"
			}
		} else {
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
