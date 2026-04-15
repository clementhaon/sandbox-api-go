package mocks

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/models"
)

// --- AuthService Mock ---

type MockAuthService struct {
	RegisterFn func(ctx context.Context, req models.RegisterRequest) (models.User, string, error)
	LoginFn    func(ctx context.Context, req models.LoginRequest) (models.User, string, error)
}

func (m *MockAuthService) Register(ctx context.Context, req models.RegisterRequest) (models.User, string, error) {
	return m.RegisterFn(ctx, req)
}
func (m *MockAuthService) Login(ctx context.Context, req models.LoginRequest) (models.User, string, error) {
	return m.LoginFn(ctx, req)
}

// --- UserService Mock ---

type MockUserService struct {
	ListFn         func(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error)
	GetByIDFn      func(ctx context.Context, id int) (models.UserResponse, error)
	CreateFn       func(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error)
	UpdateFn       func(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error)
	UpdateStatusFn func(ctx context.Context, id int, status string) (models.UserResponse, error)
	DeleteFn       func(ctx context.Context, id int) error
}

func (m *MockUserService) List(ctx context.Context, params models.UserListParams) (models.UsersListResponse, error) {
	return m.ListFn(ctx, params)
}
func (m *MockUserService) GetByID(ctx context.Context, id int) (models.UserResponse, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockUserService) Create(ctx context.Context, req models.CreateUserRequest) (models.UserResponse, error) {
	return m.CreateFn(ctx, req)
}
func (m *MockUserService) Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.UserResponse, error) {
	return m.UpdateFn(ctx, id, req)
}
func (m *MockUserService) UpdateStatus(ctx context.Context, id int, status string) (models.UserResponse, error) {
	return m.UpdateStatusFn(ctx, id, status)
}
func (m *MockUserService) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}

// --- ProfileService Mock ---

type MockProfileService struct {
	GetProfileFn    func(ctx context.Context, userID int) (models.User, error)
	UpdateProfileFn func(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error)
}

func (m *MockProfileService) GetProfile(ctx context.Context, userID int) (models.User, error) {
	return m.GetProfileFn(ctx, userID)
}
func (m *MockProfileService) UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (models.User, error) {
	return m.UpdateProfileFn(ctx, userID, req)
}

// --- TaskService Mock ---

type MockTaskService struct {
	GetBoardFn func(ctx context.Context) (models.BoardResponse, error)
	ListFn     func(ctx context.Context, columnID *int) ([]models.Task, error)
	GetByIDFn  func(ctx context.Context, id int) (models.Task, error)
	CreateFn   func(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error)
	UpdateFn   func(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error)
	MoveFn     func(ctx context.Context, id int, req models.MoveTaskRequest) (models.Task, error)
	ReorderFn  func(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error)
	DeleteFn   func(ctx context.Context, id int) error
}

func (m *MockTaskService) GetBoard(ctx context.Context) (models.BoardResponse, error) {
	return m.GetBoardFn(ctx)
}
func (m *MockTaskService) List(ctx context.Context, columnID *int) ([]models.Task, error) {
	return m.ListFn(ctx, columnID)
}
func (m *MockTaskService) GetByID(ctx context.Context, id int) (models.Task, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockTaskService) Create(ctx context.Context, userID int, req models.CreateTaskRequest) (models.Task, error) {
	return m.CreateFn(ctx, userID, req)
}
func (m *MockTaskService) Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
	return m.UpdateFn(ctx, id, req)
}
func (m *MockTaskService) Move(ctx context.Context, id int, req models.MoveTaskRequest) (models.Task, error) {
	return m.MoveFn(ctx, id, req)
}
func (m *MockTaskService) Reorder(ctx context.Context, columnID int, taskIDs []int) ([]models.Task, error) {
	return m.ReorderFn(ctx, columnID, taskIDs)
}
func (m *MockTaskService) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}

// --- ColumnService Mock ---

type MockColumnService struct {
	ListFn    func(ctx context.Context) ([]models.Column, error)
	CreateFn  func(ctx context.Context, req models.CreateColumnRequest) (models.Column, error)
	UpdateFn  func(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error)
	DeleteFn  func(ctx context.Context, id int) error
	ReorderFn func(ctx context.Context, columnIDs []int) ([]models.Column, error)
}

func (m *MockColumnService) List(ctx context.Context) ([]models.Column, error) {
	return m.ListFn(ctx)
}
func (m *MockColumnService) Create(ctx context.Context, req models.CreateColumnRequest) (models.Column, error) {
	return m.CreateFn(ctx, req)
}
func (m *MockColumnService) Update(ctx context.Context, id int, req models.UpdateColumnRequest) (models.Column, error) {
	return m.UpdateFn(ctx, id, req)
}
func (m *MockColumnService) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockColumnService) Reorder(ctx context.Context, columnIDs []int) ([]models.Column, error) {
	return m.ReorderFn(ctx, columnIDs)
}

// --- TimeEntryService Mock ---

type MockTimeEntryService struct {
	ListFn   func(ctx context.Context, taskID int) ([]models.TimeEntry, error)
	CreateFn func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
	DeleteFn func(ctx context.Context, id int) error
}

func (m *MockTimeEntryService) List(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
	return m.ListFn(ctx, taskID)
}
func (m *MockTimeEntryService) Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
	return m.CreateFn(ctx, userID, req)
}
func (m *MockTimeEntryService) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}

// --- NotificationService Mock ---

type MockNotificationService struct {
	ListFn        func(ctx context.Context, userID int) ([]models.Notification, error)
	MarkReadFn    func(ctx context.Context, userID int, notificationIDs []int) (int, error)
	MarkAllReadFn func(ctx context.Context, userID int) (int64, error)
	DeleteFn      func(ctx context.Context, userID int, id int) error
	CreateFn      func(ctx context.Context, userID int, notifType, title, message string, data models.NotificationData) error
}

func (m *MockNotificationService) List(ctx context.Context, userID int) ([]models.Notification, error) {
	return m.ListFn(ctx, userID)
}
func (m *MockNotificationService) MarkRead(ctx context.Context, userID int, notificationIDs []int) (int, error) {
	return m.MarkReadFn(ctx, userID, notificationIDs)
}
func (m *MockNotificationService) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	return m.MarkAllReadFn(ctx, userID)
}
func (m *MockNotificationService) Delete(ctx context.Context, userID int, id int) error {
	return m.DeleteFn(ctx, userID, id)
}
func (m *MockNotificationService) Create(ctx context.Context, userID int, notifType, title, message string, data models.NotificationData) error {
	return m.CreateFn(ctx, userID, notifType, title, message, data)
}

// --- MediaService Mock ---

type MockMediaService struct {
	GetPresignedUploadURLFn   func(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error)
	ConfirmUploadFn           func(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error)
	ListUserMediaFn           func(ctx context.Context, userID int, page int) (models.MediaListResponse, error)
	GetByIDFn                 func(ctx context.Context, userID int, mediaID int) (models.Media, error)
	GetPresignedDownloadURLFn func(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error)
	DeleteFn                  func(ctx context.Context, userID int, mediaID int) error
}

func (m *MockMediaService) GetPresignedUploadURL(ctx context.Context, userID int, filename, mimeType string) (models.PresignedUploadURLResponse, error) {
	return m.GetPresignedUploadURLFn(ctx, userID, filename, mimeType)
}
func (m *MockMediaService) ConfirmUpload(ctx context.Context, userID int, objectKey, originalFilename, mimeType, bucketName string) (models.Media, error) {
	return m.ConfirmUploadFn(ctx, userID, objectKey, originalFilename, mimeType, bucketName)
}
func (m *MockMediaService) ListUserMedia(ctx context.Context, userID int, page int) (models.MediaListResponse, error) {
	return m.ListUserMediaFn(ctx, userID, page)
}
func (m *MockMediaService) GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error) {
	return m.GetByIDFn(ctx, userID, mediaID)
}
func (m *MockMediaService) GetPresignedDownloadURL(ctx context.Context, userID int, mediaID int) (models.PresignedDownloadURLResponse, error) {
	return m.GetPresignedDownloadURLFn(ctx, userID, mediaID)
}
func (m *MockMediaService) Delete(ctx context.Context, userID int, mediaID int) error {
	return m.DeleteFn(ctx, userID, mediaID)
}
