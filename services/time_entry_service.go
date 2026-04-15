package services

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/errors"
	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
)

type TimeEntryService interface {
	List(ctx context.Context, taskID int) ([]models.TimeEntry, error)
	Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
	Delete(ctx context.Context, id int) error
}

type timeEntryService struct {
	timeEntryRepo repository.TimeEntryRepository
	txManager     database.Transactor
}

func NewTimeEntryService(timeEntryRepo repository.TimeEntryRepository, txManager database.Transactor) TimeEntryService {
	return &timeEntryService{timeEntryRepo: timeEntryRepo, txManager: txManager}
}

func (s *timeEntryService) List(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
	return s.timeEntryRepo.List(ctx, taskID)
}

func (s *timeEntryService) Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
	if req.TaskID == 0 {
		return models.TimeEntry{}, errors.NewBadRequestError("taskId is required")
	}
	if req.StartTime.IsZero() {
		return models.TimeEntry{}, errors.NewBadRequestError("startTime is required")
	}
	if req.Duration <= 0 {
		return models.TimeEntry{}, errors.NewBadRequestError("duration must be positive")
	}

	exists, err := s.timeEntryRepo.TaskExists(ctx, req.TaskID)
	if err != nil {
		return models.TimeEntry{}, err
	}
	if !exists {
		return models.TimeEntry{}, errors.NewNotFoundError("Task not found")
	}

	var entry models.TimeEntry
	err = s.txManager.WithTransaction(ctx, func(q database.Querier) error {
		txRepo := s.timeEntryRepo.WithQuerier(q)

		var createErr error
		entry, createErr = txRepo.Create(ctx, userID, req)
		if createErr != nil {
			return createErr
		}

		durationMinutes := req.Duration / 60
		return txRepo.AddTrackedTime(ctx, req.TaskID, durationMinutes)
	})
	if err != nil {
		return models.TimeEntry{}, err
	}

	logger.InfoContext(ctx, "Time entry created", map[string]interface{}{
		"entry_id": entry.ID,
		"task_id":  entry.TaskID,
		"duration": entry.Duration,
		"user_id":  userID,
	})

	return entry, nil
}

func (s *timeEntryService) Delete(ctx context.Context, id int) error {
	taskID, duration, err := s.timeEntryRepo.GetTaskIDAndDuration(ctx, id)
	if err != nil {
		return err
	}

	return s.txManager.WithTransaction(ctx, func(q database.Querier) error {
		txRepo := s.timeEntryRepo.WithQuerier(q)

		if err := txRepo.Delete(ctx, id); err != nil {
			return err
		}

		durationMinutes := duration / 60
		return txRepo.SubtractTrackedTime(ctx, taskID, durationMinutes)
	})
}
