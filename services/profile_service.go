package services

import (
	"context"
	"database/sql"

	"github.com/clementhaon/sandbox-api-go/logger"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID int) (models.User, error)
	UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error)
}

type profileService struct {
	userRepo repository.UserRepository
}

func NewProfileService(userRepo repository.UserRepository) ProfileService {
	return &profileService{userRepo: userRepo}
}

func (s *profileService) GetProfile(ctx context.Context, userID int) (models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	logger.InfoContext(ctx, "User profile retrieved", map[string]interface{}{
		"user_id": user.ID,
	})
	return user, nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error) {
	var firstName, lastName, avatarURL sql.NullString

	if req.FirstName != nil {
		firstName = sql.NullString{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		lastName = sql.NullString{String: *req.LastName, Valid: true}
	}
	if req.AvatarURL != nil {
		avatarURL = sql.NullString{String: *req.AvatarURL, Valid: true}
	}

	if err := s.userRepo.UpdateProfile(ctx, userID, firstName, lastName, avatarURL); err != nil {
		return models.User{}, err
	}

	updatedUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	logger.InfoContext(ctx, "User profile updated successfully", map[string]interface{}{
		"user_id": updatedUser.ID,
	})
	return updatedUser, nil
}
