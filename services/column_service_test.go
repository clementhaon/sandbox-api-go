package services

import (
	"context"
	"testing"
	"time"

	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestColumnService_Create_DefaultColor(t *testing.T) {
	var createdColor string
	repo := &mocks.MockColumnRepository{
		GetMaxOrderFn: func(ctx context.Context) (int, error) {
			return 1, nil
		},
		CreateFn: func(ctx context.Context, title, color string, order int) (models.Column, error) {
			createdColor = color
			return models.Column{ID: 1, Title: title, Color: color, Order: order}, nil
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})
	col, err := svc.Create(context.Background(), models.CreateColumnRequest{Title: "New Column"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdColor != "#2196F3" {
		t.Errorf("expected default color #2196F3, got %s", createdColor)
	}
	if col.Order != 2 {
		t.Errorf("expected order 2, got %d", col.Order)
	}
}

func TestColumnService_Create_CustomColor(t *testing.T) {
	repo := &mocks.MockColumnRepository{
		GetMaxOrderFn: func(ctx context.Context) (int, error) { return 0, nil },
		CreateFn: func(ctx context.Context, title, color string, order int) (models.Column, error) {
			return models.Column{ID: 1, Title: title, Color: color, Order: order}, nil
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})
	col, err := svc.Create(context.Background(), models.CreateColumnRequest{Title: "Red", Color: "#FF0000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Color != "#FF0000" {
		t.Errorf("expected color #FF0000, got %s", col.Color)
	}
}

func TestColumnService_Update_PartialFields(t *testing.T) {
	repo := &mocks.MockColumnRepository{
		GetByIDFn: func(ctx context.Context, id int) (models.Column, error) {
			return models.Column{ID: 1, Title: "Old", Color: "#000", CreatedAt: time.Now()}, nil
		},
		UpdateFn: func(ctx context.Context, id int, title, color string) (models.Column, error) {
			return models.Column{ID: id, Title: title, Color: color}, nil
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})

	// Only update title, keep existing color
	col, err := svc.Update(context.Background(), 1, models.UpdateColumnRequest{Title: "New Title"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col.Title != "New Title" {
		t.Errorf("expected 'New Title', got '%s'", col.Title)
	}
	if col.Color != "#000" {
		t.Errorf("expected existing color '#000', got '%s'", col.Color)
	}
}

func TestColumnService_Delete_LastColumn(t *testing.T) {
	repo := &mocks.MockColumnRepository{
		GetFirstOtherColumnFn: func(ctx context.Context, excludeID int) (int, error) {
			return 0, errors.NewBadRequestError("Cannot delete the last column")
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})
	err := svc.Delete(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for deleting last column")
	}
	appErr, ok := errors.IsAppError(err)
	if !ok {
		t.Fatal("expected AppError")
	}
	if appErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
}

func TestColumnService_Delete_Success(t *testing.T) {
	moveTasksCalled := false
	deleteCalled := false
	reorderCalled := false

	repo := &mocks.MockColumnRepository{
		GetFirstOtherColumnFn: func(ctx context.Context, excludeID int) (int, error) {
			return 2, nil
		},
		MoveTasksToColumnFn: func(ctx context.Context, from, to int) error {
			moveTasksCalled = true
			if from != 1 || to != 2 {
				t.Errorf("expected move from 1 to 2, got from %d to %d", from, to)
			}
			return nil
		},
		DeleteFn: func(ctx context.Context, id int) error {
			deleteCalled = true
			return nil
		},
		ReorderAfterDeleteFn: func(ctx context.Context) error {
			reorderCalled = true
			return nil
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})
	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !moveTasksCalled {
		t.Error("expected MoveTasksToColumn to be called")
	}
	if !deleteCalled {
		t.Error("expected Delete to be called")
	}
	if !reorderCalled {
		t.Error("expected ReorderAfterDelete to be called")
	}
}

func TestColumnService_Reorder(t *testing.T) {
	reorderCalled := false
	repo := &mocks.MockColumnRepository{
		ReorderFn: func(ctx context.Context, columnIDs []int) error {
			reorderCalled = true
			return nil
		},
		ListFn: func(ctx context.Context) ([]models.Column, error) {
			return []models.Column{{ID: 2}, {ID: 1}}, nil
		},
	}

	svc := NewColumnService(repo, &mocks.MockTransactor{})
	cols, err := svc.Reorder(context.Background(), []int{2, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reorderCalled {
		t.Error("expected Reorder to be called")
	}
	if len(cols) != 2 {
		t.Errorf("expected 2 columns, got %d", len(cols))
	}
}
