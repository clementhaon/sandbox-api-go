package services

import (
	"context"

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
}

func NewTimeEntryService(timeEntryRepo repository.TimeEntryRepository) TimeEntryService {
	return &timeEntryService{timeEntryRepo: timeEntryRepo}
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

	entry, err := s.timeEntryRepo.Create(ctx, userID, req)
	if err != nil {
		return models.TimeEntry{}, err
	}

	durationMinutes := req.Duration / 60
	if err := s.timeEntryRepo.AddTrackedTime(ctx, req.TaskID, durationMinutes); err != nil {
		logger.WarnContext(ctx, "Error updating task tracked_time", map[string]interface{}{
			"error":   err.Error(),
			"task_id": req.TaskID,
		})
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

	if err := s.timeEntryRepo.Delete(ctx, id); err != nil {
		return err
	}

	durationMinutes := duration / 60
	if err := s.timeEntryRepo.SubtractTrackedTime(ctx, taskID, durationMinutes); err != nil {
		logger.WarnContext(ctx, "Error updating task tracked_time after delete", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
	}

	return nil
}
