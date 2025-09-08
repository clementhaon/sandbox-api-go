package database

import (
	"database/sql"
	"fmt"
	"log"
	"sandbox-api-go/config"
	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initialise la connexion à la base de données
func InitDB() error {
	// Récupération des variables d'environnement
	dbHost := config.GetEnv("DB_HOST", "localhost")
	dbPort := config.GetEnv("DB_PORT", "5432")
	dbUser := config.GetEnv("DB_USER", "postgres")
	dbPassword := config.GetEnv("DB_PASSWORD", "postgres123")
	dbName := config.GetEnv("DB_NAME", "sandbox_api")
	dbSSLMode := config.GetEnv("DB_SSLMODE", "disable")

	// Construction de la chaîne de connexion
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Connexion à la base de données
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture de la connexion: %v", err)
	}

	// Test de la connexion
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("erreur lors du test de connexion: %v", err)
	}

	// Configuration de la connexion
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)

	log.Println("✅ Connexion à PostgreSQL établie avec succès")
	return nil
}

// CloseDB ferme la connexion à la base de données
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

