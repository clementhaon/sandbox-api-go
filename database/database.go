package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/clementhaon/sandbox-api-go/config"
	_ "github.com/lib/pq"
	"log"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB() error {
	// Load environment variables
	dbHost := config.GetEnv("DB_HOST", "localhost")
	dbPort := config.GetEnv("DB_PORT", "5432")
	dbUser := config.GetEnv("DB_USER", "postgres")
	dbPassword := config.GetEnv("DB_PASSWORD", "postgres123")
	dbName := config.GetEnv("DB_NAME", "sandbox_api")
	dbSSLMode := config.GetEnv("DB_SSLMODE", "disable")

	// Build the connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Connect to the database
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("error opening database connection: %v", err)
	}

	// Test the connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error testing database connection: %v", err)
	}

	// Configure the connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(1 * time.Minute)

	log.Println("✅ PostgreSQL connection established successfully")

	// Run migrations automatically
	if err := RunMigrations(DB); err != nil {
		return fmt.Errorf("error running migrations: %v", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
