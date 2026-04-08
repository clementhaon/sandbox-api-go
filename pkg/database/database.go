package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/clementhaon/sandbox-api-go/pkg/config"
	_ "github.com/lib/pq"
)

// InitDB creates and returns a database connection using environment variables.
func InitDB() (*sql.DB, error) {
	dbHost := config.GetEnv("DB_HOST", "localhost")
	dbPort := config.GetEnv("DB_PORT", "5432")
	dbUser := config.GetEnv("DB_USER", "postgres")
	dbPassword := config.GetEnv("DB_PASSWORD", "postgres123")
	dbName := config.GetEnv("DB_NAME", "sandbox_api")
	dbSSLMode := config.GetEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("error testing database connection: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	log.Println("PostgreSQL connection established successfully")
	return db, nil
}

// RunMigrations runs database migrations from the given path.
func RunMigrations(db *sql.DB, migrationsPath string) error {
	// Import is handled by the caller since migrate requires file:// source
	// This is a placeholder — each service runs its own migrations in main.go
	return nil
}
