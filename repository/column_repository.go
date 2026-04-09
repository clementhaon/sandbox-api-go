package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
)

type ColumnRepository interface {
	List(ctx context.Context) ([]models.Column, error)
	GetByID(ctx context.Context, id int) (models.Column, error)
	GetMaxOrder(ctx context.Context) (int, error)
	Create(ctx context.Context, title, color string, order int) (models.Column, error)
	Update(ctx context.Context, id int, title, color string) (models.Column, error)
	GetFirstOtherColumn(ctx context.Context, excludeID int) (int, error)
	MoveTasksToColumn(ctx context.Context, fromColumnID, toColumnID int) error
	Delete(ctx context.Context, id int) error
	ReorderAfterDelete(ctx context.Context) error
	Reorder(ctx context.Context, columnIDs []int) error
}

type postgresColumnRepo struct {
	db *sql.DB
}

func NewPostgresColumnRepository(db *sql.DB) ColumnRepository {
	return &postgresColumnRepo{db: db}
}

func scanColumn(row interface{ Scan(...any) error }) (models.Column, error) {
	var c models.Column
	err := row.Scan(&c.ID, &c.Title, &c.Order, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *postgresColumnRepo) List(ctx context.Context) ([]models.Column, error) {
	startTime := time.Now()
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, "order", color, created_at, updated_at FROM columns ORDER BY "order" ASC`)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error querying columns", err)
		return nil, errors.NewDatabaseError().WithCause(err)
	}
	defer rows.Close()

	columns := []models.Column{}
	for rows.Next() {
		c, err := scanColumn(rows)
		if err != nil {
			logger.ErrorContext(ctx, "Error scanning column row", err)
			return nil, errors.NewDatabaseError().WithCause(err)
		}
		columns = append(columns, c)
	}
	return columns, nil
}

func (r *postgresColumnRepo) GetByID(ctx context.Context, id int) (models.Column, error) {
	startTime := time.Now()
	c, err := scanColumn(r.db.QueryRowContext(ctx,
		`SELECT id, title, "order", color, created_at, updated_at FROM columns WHERE id = $1`, id))
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return models.Column{}, errors.NewNotFoundError("Column not found")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error fetching column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}
	return c, nil
}

func (r *postgresColumnRepo) GetMaxOrder(ctx context.Context) (int, error) {
	var maxOrder int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX("order"), -1) FROM columns`).Scan(&maxOrder)
	logger.LogDatabaseOperation(ctx, "SELECT MAX", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error getting max order", err)
		return 0, errors.NewDatabaseError().WithCause(err)
	}
	return maxOrder, nil
}

func (r *postgresColumnRepo) Create(ctx context.Context, title, color string, order int) (models.Column, error) {
	startTime := time.Now()
	c, err := scanColumn(r.db.QueryRowContext(ctx,
		`INSERT INTO columns (title, "order", color) VALUES ($1, $2, $3)
		RETURNING id, title, "order", color, created_at, updated_at`,
		title, order, color,
	))
	logger.LogDatabaseOperation(ctx, "INSERT", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error creating column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}
	return c, nil
}

func (r *postgresColumnRepo) Update(ctx context.Context, id int, title, color string) (models.Column, error) {
	startTime := time.Now()
	c, err := scanColumn(r.db.QueryRowContext(ctx,
		`UPDATE columns SET title = $1, color = $2, updated_at = NOW() WHERE id = $3
		RETURNING id, title, "order", color, created_at, updated_at`,
		title, color, id,
	))
	logger.LogDatabaseOperation(ctx, "UPDATE", "columns", time.Since(startTime), err)

	if err != nil {
		logger.ErrorContext(ctx, "Error updating column", err)
		return models.Column{}, errors.NewDatabaseError().WithCause(err)
	}
	return c, nil
}

func (r *postgresColumnRepo) GetFirstOtherColumn(ctx context.Context, excludeID int) (int, error) {
	var id int
	startTime := time.Now()
	err := r.db.QueryRowContext(ctx, `SELECT id FROM columns WHERE id != $1 ORDER BY "order" ASC LIMIT 1`, excludeID).Scan(&id)
	logger.LogDatabaseOperation(ctx, "SELECT", "columns", time.Since(startTime), err)

	if err == sql.ErrNoRows {
		return 0, errors.NewBadRequestError("Cannot delete the last column")
	}
	if err != nil {
		logger.ErrorContext(ctx, "Error finding first column", err)
		return 0, errors.NewDatabaseError().WithCause(err)
	}
	return id, nil
}

func (r *postgresColumnRepo) MoveTasksToColumn(ctx context.Context, fromColumnID, toColumnID int) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET column_id = $1, updated_at = NOW() WHERE column_id = $2`, toColumnID, fromColumnID)
	logger.LogDatabaseOperation(ctx, "UPDATE", "tasks", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error moving tasks", err)
		return errors.NewDatabaseError().WithCause(err)
	}
	return nil
}

func (r *postgresColumnRepo) Delete(ctx context.Context, id int) error {
	startTime := time.Now()
	result, err := r.db.ExecContext(ctx, `DELETE FROM columns WHERE id = $1`, id)
	logger.LogDatabaseOperation(ctx, "DELETE", "columns", time.Since(startTime), err)
	if err != nil {
		logger.ErrorContext(ctx, "Error deleting column", err)
		return errors.NewDatabaseError().WithCause(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.NewNotFoundError("Column not found")
	}
	return nil
}

func (r *postgresColumnRepo) ReorderAfterDelete(ctx context.Context) error {
	startTime := time.Now()
	_, err := r.db.ExecContext(ctx, `
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
	return err
}

func (r *postgresColumnRepo) Reorder(ctx context.Context, columnIDs []int) error {
	for i, columnID := range columnIDs {
		startTime := time.Now()
		result, err := r.db.ExecContext(ctx, `UPDATE columns SET "order" = $1, updated_at = NOW() WHERE id = $2`, i, columnID)
		logger.LogDatabaseOperation(ctx, "UPDATE", "columns", time.Since(startTime), err)

		if err != nil {
			logger.ErrorContext(ctx, "Error updating column order", err)
			return errors.NewDatabaseError().WithCause(err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.NewNotFoundError("Column not found: " + strconv.Itoa(columnID))
		}
	}
	return nil
}
