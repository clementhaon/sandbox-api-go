package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/errors"
	"github.com/clementhaon/sandbox-api-go/pkg/logger"
	"github.com/clementhaon/sandbox-api-go/pkg/models"
)

type ColumnService interface {
	List(ctx context.Context) ([]models.Column, error)
	Create(ctx context.Context, req models.CreateColumnRequest) (models.Column, error)
	Update(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error)
	Delete(ctx context.Context, id int) error
	Reorder(ctx context.Context, columnIDs []int) ([]models.Column, error)
}

type columnService struct {
	db *sql.DB
}

func NewColumnService(db *sql.DB) ColumnService {
	return &columnService{db: db}
}

func (s *columnService) List(ctx context.Context) ([]models.Column, error) {
	startTime := time.Now()
	rows, err := s.db.Query(`SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error querying columns", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	columns := []models.Column{}
	for rows.Next() {
		var c models.Column
		err := rows.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning column row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}

	return columns, nil
}

func (s *columnService) Create(ctx context.Context, req models.CreateColumnRequest) (models.Column, error) {
	if req.Color == "" {
		req.Color = "#2196F3"
	}

	var maxOrder int
	startTime := time.Now()
	err := s.db.QueryRow(`SELECT COALESCE(MAX("order"), -1) FROM columns`).Scan(&maxOrder)
	logger.LogDatabaseOperation(ctx, "SELECT MAX", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error getting max order", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}

	var c models.Column
	startTime = time.Now()
	err = s.db.QueryRow(
		`INSERT INTO columns (title, "order", color) VALUES ($1, $2, $3)
		RETURNING id, title, "order", color, created_at, updated_at`,
		req.Title, maxOrder+1, req.Color,
	).Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "INSERT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}

	return c, nil
}

func (s *columnService) Update(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error) {
	var existing models.Column
	startTime := time.Now()
	err := s.db.QueryRow(`SELECT id, title, "order", color, created_at, updated_at FROM columns WHERE id = $1`, id).
		Scan(&existing.ID, &existing.Title, &existing.Order, &existing.Color, &existing.CreatedAt, &existing.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Column{}, errors.NewNotFoundError("Column not found")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error fetching column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Color != "" {
		existing.Color = req.Color
	}

	var c models.Column
	startTime = time.Now()
	err = s.db.QueryRow(
		`UPDATE columns SET title = $1, color = $2, updated_at = NOW() WHERE id = $3
		RETURNING id, title, "order", color, created_at, updated_at`,
		existing.Title, existing.Color, id,
	).Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	logger.LogDatabaseOperation(ctx, "UPDATE", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}

	return c, nil
}

func (s *columnService) Delete(ctx context.Context, id int) error {
	var firstColumnID int
	startTime := time.Now()
	err := s.db.QueryRow(`SELECT id FROM columns WHERE id != $1 ORDER BY "order" ASC LIMIT 1`, id).Scan(&firstColumnID)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return errors.NewBadRequestError("Cannot delete the last column")
	} else if err != nil {
		logger.ErrorContext(ctx, "Error finding first column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	startTime = time.Now()
	_, err = s.db.Exec(`UPDATE tasks SET column_id = $1, updated_at = NOW() WHERE column_id = $2`, firstColumnID, id)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error moving tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	startTime = time.Now()
	result, err := s.db.Exec(`DELETE FROM columns WHERE id = $1`, id)
	logger.LogDatabaseOperation(ctx, "DELETE", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error deleting column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Column not found")
	}

	startTime = time.Now()
	_, err = s.db.Exec(`
		WITH ordered AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY "order") - 1 as new_order
			FROM columns
		)
		UPDATE columns SET "order" = ordered.new_order
		FROM ordered WHERE columns.id = ordered.id
	`)
	logger.LogDatabaseOperation(ctx, "UPDATE", "columns", time.Since(startTime), err)
	if err != nil {
		logger.WarnContext(ctx, "Error reordering columns after delete", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return nil
}

func (s *columnService) Reorder(ctx context.Context, columnIDs []int) ([]models.Column, error) {
	for i, columnID := range columnIDs {
		startTime := time.Now()
		result, err := s.db.Exec(`UPDATE columns SET "order" = $1, updated_at = NOW() WHERE id = $2`, i, columnID)
		logger.LogDatabaseOperation(ctx, "UPDATE", "columns", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(ctx, "Error updating column order", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return nil, errors.NewNotFoundError("Column not found: " + string(rune(columnID)))
		}
	}

	return s.List(ctx)
}
