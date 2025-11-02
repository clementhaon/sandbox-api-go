package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations exécute les migrations de base de données
func RunMigrations(db *sql.DB) error {
	// Créer le driver postgres pour migrate
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("erreur lors de la création du driver postgres: %v", err)
	}

	// Créer l'instance de migration
	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("erreur lors de l'initialisation des migrations: %v", err)
	}

	// Exécuter les migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("erreur lors de l'exécution des migrations: %v", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("erreur lors de la récupération de la version: %v", err)
	}

	if err == migrate.ErrNilVersion {
		log.Println("✅ Base de données initialisée (aucune migration)")
	} else {
		log.Printf("✅ Migrations appliquées avec succès (version: %d, dirty: %t)\n", version, dirty)
	}

	return nil
}

// RollbackMigration rollback la dernière migration
func RollbackMigration(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("erreur lors de la création du driver postgres: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("erreur lors de l'initialisation des migrations: %v", err)
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("erreur lors du rollback: %v", err)
	}

	log.Println("✅ Rollback effectué avec succès")
	return nil
}

// GetMigrationVersion retourne la version actuelle de la migration
func GetMigrationVersion(db *sql.DB) (uint, bool, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("erreur lors de la création du driver postgres: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return 0, false, fmt.Errorf("erreur lors de l'initialisation des migrations: %v", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return 0, false, err
	}

	return version, dirty, nil
}
