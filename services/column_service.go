package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
)

type ColumnService interface {
	List(ctx context.Context) ([]models.Column, error)
	Create(ctx context.Context, req models.CreateColumnRequest) (models.Column, error)
	Update(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error)
	Delete(ctx context.Context, id int) error
	Reorder(ctx context.Context, columnIDs []int) ([]models.Column, error)
}

type columnService struct {
	columnRepo repository.ColumnRepository
}

func NewColumnService(columnRepo repository.ColumnRepository) ColumnService {
	return &columnService{columnRepo: columnRepo}
}

func (s *columnService) List(ctx context.Context) ([]models.Column, error) {
	return s.columnRepo.List(ctx)
}

func (s *columnService) Create(ctx context.Context, req models.CreateColumnRequest) (models.Column, error) {
	if req.Color == "" {
		req.Color = "#2196F3"
	}

	maxOrder, err := s.columnRepo.GetMaxOrder(ctx)
	if err != nil {
		return models.Column{}, err
	}

	return s.columnRepo.Create(ctx, req.Title, req.Color, maxOrder+1)
}

func (s *columnService) Update(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error) {
	existing, err := s.columnRepo.GetByID(ctx, id)
	if err != nil {
		return models.Column{}, err
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Color != "" {
		existing.Color = req.Color
	}

	return s.columnRepo.Update(ctx, id, existing.Title, existing.Color)
}

func (s *columnService) Delete(ctx context.Context, id int) error {
	firstColumnID, err := s.columnRepo.GetFirstOtherColumn(ctx, id)
	if err != nil {
		return err
	}

	if err := s.columnRepo.MoveTasksToColumn(ctx, id, firstColumnID); err != nil {
		return err
	}

	if err := s.columnRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.columnRepo.ReorderAfterDelete(ctx)
	return nil
}

func (s *columnService) Reorder(ctx context.Context, columnIDs []int) ([]models.Column, error) {
	if err := s.columnRepo.Reorder(ctx, columnIDs); err != nil {
		return nil, err
	}
	return s.columnRepo.List(ctx)
}
