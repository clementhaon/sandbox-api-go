package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/clementhaon/sandbox-api-go/mocks"
	"github.com/clementhaon/sandbox-api-go/models"
)

func TestProfileService_GetProfile(t *testing.T) {
	tests := []struct {
		name      string
		userID    int
		getByIDFn func(ctx context.Context, id int) (models.User, error)
		wantErr   bool
		wantID    int
	}{
		{
			name:   "success",
			userID: 1,
			getByIDFn: func(ctx context.Context, id int) (models.User, error) {
				return models.User{ID: id, Username: "alice"}, nil
			},
			wantID: 1,
		},
		{
			name:   "not found",
			userID: 999,
			getByIDFn: func(ctx context.Context, id int) (models.User, error) {
				return models.User{}, fmt.Errorf("user not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockUserRepository{GetByIDFn: tt.getByIDFn}
			svc := NewProfileService(repo)

			user, err := svc.GetProfile(context.Background(), tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user.ID != tt.wantID {
				t.Errorf("expected user ID %d, got %d", tt.wantID, user.ID)
			}
		})
	}
}

func TestProfileService_UpdateProfile(t *testing.T) {
	firstName := "Alice"
	tests := []struct {
		name            string
		userID          int
		req             models.UpdateProfileRequest
		updateProfileFn func(ctx context.Context, userID int, firstName, lastName, avatarURL sql.NullString) error
		getByIDFn       func(ctx context.Context, id int) (models.User, error)
		wantErr         bool
	}{
		{
			name:   "success",
			userID: 1,
			req:    models.UpdateProfileRequest{FirstName: &firstName},
			updateProfileFn: func(ctx context.Context, userID int, fn, ln, av sql.NullString) error {
				if !fn.Valid || fn.String != "Alice" {
					return fmt.Errorf("expected firstName Alice")
				}
				return nil
			},
			getByIDFn: func(ctx context.Context, id int) (models.User, error) {
				return models.User{ID: id, Username: "alice", FirstName: sql.NullString{String: "Alice", Valid: true}}, nil
			},
		},
		{
			name:   "repo error on update",
			userID: 1,
			req:    models.UpdateProfileRequest{FirstName: &firstName},
			updateProfileFn: func(ctx context.Context, userID int, fn, ln, av sql.NullString) error {
				return fmt.Errorf("db error")
			},
			wantErr: true,
		},
		{
			name:   "repo error on get after update",
			userID: 1,
			req:    models.UpdateProfileRequest{FirstName: &firstName},
			updateProfileFn: func(ctx context.Context, userID int, fn, ln, av sql.NullString) error {
				return nil
			},
			getByIDFn: func(ctx context.Context, id int) (models.User, error) {
				return models.User{}, fmt.Errorf("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mocks.MockUserRepository{
				UpdateProfileFn: tt.updateProfileFn,
				GetByIDFn:       tt.getByIDFn,
			}
			svc := NewProfileService(repo)

			user, err := svc.UpdateProfile(context.Background(), tt.userID, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if user.ID != tt.userID {
				t.Errorf("expected user ID %d, got %d", tt.userID, user.ID)
			}
		})
	}
}
