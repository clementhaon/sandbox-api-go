package mocks

import (
	"context"
	"database/sql"

	"github.com/clementhaon/sandbox-api-go/database"
	"github.com/clementhaon/sandbox-api-go/models"
	"github.com/clementhaon/sandbox-api-go/repository"
)

// --- UserRepository Mock ---

type MockUserRepository struct {
	ExistsByUsernameOrEmailFn func(ctx context.Context, username, email string) (bool, error)
	CreateAuthFn              func(ctx context.Context, username, email, hashedPassword string) (models.User, error)
	FindByEmailWithPasswordFn func(ctx context.Context, email string) (models.User, string, error)
	UpdateLastLoginFn         func(ctx context.Context, userID int) error
	ListFn                    func(ctx context.Context, params models.UserListParams) ([]models.User, int, error)
	GetByIDFn                 func(ctx context.Context, id int) (models.User, error)
	ExistsFn                  func(ctx context.Context, id int) (bool, error)
	CreateFn                  func(ctx context.Context, username, email, hashedPassword, firstName, lastName, role string) (models.User, error)
	UpdateFn                  func(ctx context.Context, id int, req models.UpdateUserRequest) (models.User, error)
	UpdateStatusFn            func(ctx context.Context, id int, isActive bool) (models.User, error)
	DeleteFn                  func(ctx context.Context, id int) error
	UpdateProfileFn           func(ctx context.Context, userID int, firstName, lastName, avatarURL sql.NullString) error
}

func (m *MockUserRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	return m.ExistsByUsernameOrEmailFn(ctx, username, email)
}
func (m *MockUserRepository) CreateAuth(ctx context.Context, username, email, hashedPassword string) (models.User, error) {
	return m.CreateAuthFn(ctx, username, email, hashedPassword)
}
func (m *MockUserRepository) FindByEmailWithPassword(ctx context.Context, email string) (models.User, string, error) {
	return m.FindByEmailWithPasswordFn(ctx, email)
}
func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID int) error {
	if m.UpdateLastLoginFn != nil {
		return m.UpdateLastLoginFn(ctx, userID)
	}
	return nil
}
func (m *MockUserRepository) List(ctx context.Context, params models.UserListParams) ([]models.User, int, error) {
	return m.ListFn(ctx, params)
}
func (m *MockUserRepository) GetByID(ctx context.Context, id int) (models.User, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockUserRepository) Exists(ctx context.Context, id int) (bool, error) {
	return m.ExistsFn(ctx, id)
}
func (m *MockUserRepository) Create(ctx context.Context, username, email, hashedPassword, firstName, lastName, role string) (models.User, error) {
	return m.CreateFn(ctx, username, email, hashedPassword, firstName, lastName, role)
}
func (m *MockUserRepository) Update(ctx context.Context, id int, req models.UpdateUserRequest) (models.User, error) {
	return m.UpdateFn(ctx, id, req)
}
func (m *MockUserRepository) UpdateStatus(ctx context.Context, id int, isActive bool) (models.User, error) {
	return m.UpdateStatusFn(ctx, id, isActive)
}
func (m *MockUserRepository) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockUserRepository) UpdateProfile(ctx context.Context, userID int, firstName, lastName, avatarURL sql.NullString) error {
	return m.UpdateProfileFn(ctx, userID, firstName, lastName, avatarURL)
}
func (m *MockUserRepository) WithQuerier(_ database.Querier) repository.UserRepository {
	return m
}

// --- TaskRepository Mock ---

type MockTaskRepository struct {
	ListWithAssigneeFn func(ctx context.Context, columnID *int) ([]models.Task, error)
	GetByIDFn          func(ctx context.Context, id int) (models.Task, error)
	GetMaxOrderFn      func(ctx context.Context, columnID int) (int, error)
	CreateFn           func(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error)
	ExistsFn           func(ctx context.Context, id int) (bool, error)
	UpdateFn           func(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error)
	MoveFn             func(ctx context.Context, id int, columnID int, order int) (models.Task, error)
	ReorderFn          func(ctx context.Context, columnID int, taskIDs []int) error
	DeleteFn           func(ctx context.Context, id int) error
}

func (m *MockTaskRepository) ListWithAssignee(ctx context.Context, columnID *int) ([]models.Task, error) {
	return m.ListWithAssigneeFn(ctx, columnID)
}
func (m *MockTaskRepository) GetByID(ctx context.Context, id int) (models.Task, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockTaskRepository) GetMaxOrder(ctx context.Context, columnID int) (int, error) {
	return m.GetMaxOrderFn(ctx, columnID)
}
func (m *MockTaskRepository) Create(ctx context.Context, req models.CreateTaskRequest, order int, userID int) (models.Task, error) {
	return m.CreateFn(ctx, req, order, userID)
}
func (m *MockTaskRepository) Exists(ctx context.Context, id int) (bool, error) {
	return m.ExistsFn(ctx, id)
}
func (m *MockTaskRepository) Update(ctx context.Context, id int, req models.UpdateTaskRequest) (models.Task, error) {
	return m.UpdateFn(ctx, id, req)
}
func (m *MockTaskRepository) Move(ctx context.Context, id int, columnID int, order int) (models.Task, error) {
	return m.MoveFn(ctx, id, columnID, order)
}
func (m *MockTaskRepository) Reorder(ctx context.Context, columnID int, taskIDs []int) error {
	return m.ReorderFn(ctx, columnID, taskIDs)
}
func (m *MockTaskRepository) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockTaskRepository) WithQuerier(_ database.Querier) repository.TaskRepository {
	return m
}

// --- ColumnRepository Mock ---

type MockColumnRepository struct {
	ListFn               func(ctx context.Context) ([]models.Column, error)
	GetByIDFn            func(ctx context.Context, id int) (models.Column, error)
	GetMaxOrderFn        func(ctx context.Context) (int, error)
	CreateFn             func(ctx context.Context, title, color string, order int) (models.Column, error)
	UpdateFn             func(ctx context.Context, id int, title, color string) (models.Column, error)
	GetFirstOtherColumnFn func(ctx context.Context, excludeID int) (int, error)
	MoveTasksToColumnFn  func(ctx context.Context, fromColumnID, toColumnID int) error
	DeleteFn             func(ctx context.Context, id int) error
	ReorderAfterDeleteFn func(ctx context.Context) error
	ReorderFn            func(ctx context.Context, columnIDs []int) error
}

func (m *MockColumnRepository) List(ctx context.Context) ([]models.Column, error) {
	return m.ListFn(ctx)
}
func (m *MockColumnRepository) GetByID(ctx context.Context, id int) (models.Column, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockColumnRepository) GetMaxOrder(ctx context.Context) (int, error) {
	return m.GetMaxOrderFn(ctx)
}
func (m *MockColumnRepository) Create(ctx context.Context, title, color string, order int) (models.Column, error) {
	return m.CreateFn(ctx, title, color, order)
}
func (m *MockColumnRepository) Update(ctx context.Context, id int, title, color string) (models.Column, error) {
	return m.UpdateFn(ctx, id, title, color)
}
func (m *MockColumnRepository) GetFirstOtherColumn(ctx context.Context, excludeID int) (int, error) {
	return m.GetFirstOtherColumnFn(ctx, excludeID)
}
func (m *MockColumnRepository) MoveTasksToColumn(ctx context.Context, fromColumnID, toColumnID int) error {
	return m.MoveTasksToColumnFn(ctx, fromColumnID, toColumnID)
}
func (m *MockColumnRepository) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockColumnRepository) ReorderAfterDelete(ctx context.Context) error {
	if m.ReorderAfterDeleteFn != nil {
		return m.ReorderAfterDeleteFn(ctx)
	}
	return nil
}
func (m *MockColumnRepository) Reorder(ctx context.Context, columnIDs []int) error {
	return m.ReorderFn(ctx, columnIDs)
}
func (m *MockColumnRepository) WithQuerier(_ database.Querier) repository.ColumnRepository {
	return m
}

// --- TimeEntryRepository Mock ---

type MockTimeEntryRepository struct {
	ListFn                func(ctx context.Context, taskID int) ([]models.TimeEntry, error)
	TaskExistsFn          func(ctx context.Context, taskID int) (bool, error)
	CreateFn              func(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error)
	AddTrackedTimeFn      func(ctx context.Context, taskID int, durationMinutes int) error
	GetTaskIDAndDurationFn func(ctx context.Context, id int) (int, int, error)
	DeleteFn              func(ctx context.Context, id int) error
	SubtractTrackedTimeFn func(ctx context.Context, taskID int, durationMinutes int) error
}

func (m *MockTimeEntryRepository) List(ctx context.Context, taskID int) ([]models.TimeEntry, error) {
	return m.ListFn(ctx, taskID)
}
func (m *MockTimeEntryRepository) TaskExists(ctx context.Context, taskID int) (bool, error) {
	return m.TaskExistsFn(ctx, taskID)
}
func (m *MockTimeEntryRepository) Create(ctx context.Context, userID int, req models.CreateTimeEntryRequest) (models.TimeEntry, error) {
	return m.CreateFn(ctx, userID, req)
}
func (m *MockTimeEntryRepository) AddTrackedTime(ctx context.Context, taskID int, durationMinutes int) error {
	if m.AddTrackedTimeFn != nil {
		return m.AddTrackedTimeFn(ctx, taskID, durationMinutes)
	}
	return nil
}
func (m *MockTimeEntryRepository) GetTaskIDAndDuration(ctx context.Context, id int) (int, int, error) {
	return m.GetTaskIDAndDurationFn(ctx, id)
}
func (m *MockTimeEntryRepository) Delete(ctx context.Context, id int) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockTimeEntryRepository) SubtractTrackedTime(ctx context.Context, taskID int, durationMinutes int) error {
	if m.SubtractTrackedTimeFn != nil {
		return m.SubtractTrackedTimeFn(ctx, taskID, durationMinutes)
	}
	return nil
}
func (m *MockTimeEntryRepository) WithQuerier(_ database.Querier) repository.TimeEntryRepository {
	return m
}

// --- NotificationRepository Mock ---

type MockNotificationRepository struct {
	ListFn        func(ctx context.Context, userID int) ([]models.Notification, error)
	MarkReadFn    func(ctx context.Context, userID int, notificationIDs []int) error
	MarkAllReadFn func(ctx context.Context, userID int) (int64, error)
	DeleteFn      func(ctx context.Context, userID int, id int) error
	CreateFn      func(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error
}

func (m *MockNotificationRepository) List(ctx context.Context, userID int) ([]models.Notification, error) {
	return m.ListFn(ctx, userID)
}
func (m *MockNotificationRepository) MarkRead(ctx context.Context, userID int, notificationIDs []int) error {
	return m.MarkReadFn(ctx, userID, notificationIDs)
}
func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, userID int) (int64, error) {
	return m.MarkAllReadFn(ctx, userID)
}
func (m *MockNotificationRepository) Delete(ctx context.Context, userID int, id int) error {
	return m.DeleteFn(ctx, userID, id)
}
func (m *MockNotificationRepository) Create(ctx context.Context, userID int, notifType, title, message string, dataJSON []byte) error {
	return m.CreateFn(ctx, userID, notifType, title, message, dataJSON)
}
func (m *MockNotificationRepository) WithQuerier(_ database.Querier) repository.NotificationRepository {
	return m
}

// --- MediaRepository Mock ---

type MockMediaRepository struct {
	CreateFn       func(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error)
	CountFn        func(ctx context.Context, userID int) (int, error)
	ListFn         func(ctx context.Context, userID int, limit, offset int) ([]models.Media, error)
	GetByIDFn      func(ctx context.Context, userID int, mediaID int) (models.Media, error)
	GetObjectKeyFn func(ctx context.Context, userID int, mediaID int) (string, error)
	DeleteFn       func(ctx context.Context, userID int, mediaID int) error
}

func (m *MockMediaRepository) Create(ctx context.Context, userID int, objectKey, bucketName, originalFilename, mimeType string, fileSize int64) (models.Media, error) {
	return m.CreateFn(ctx, userID, objectKey, bucketName, originalFilename, mimeType, fileSize)
}
func (m *MockMediaRepository) Count(ctx context.Context, userID int) (int, error) {
	return m.CountFn(ctx, userID)
}
func (m *MockMediaRepository) List(ctx context.Context, userID int, limit, offset int) ([]models.Media, error) {
	return m.ListFn(ctx, userID, limit, offset)
}
func (m *MockMediaRepository) GetByID(ctx context.Context, userID int, mediaID int) (models.Media, error) {
	return m.GetByIDFn(ctx, userID, mediaID)
}
func (m *MockMediaRepository) GetObjectKey(ctx context.Context, userID int, mediaID int) (string, error) {
	return m.GetObjectKeyFn(ctx, userID, mediaID)
}
func (m *MockMediaRepository) Delete(ctx context.Context, userID int, mediaID int) error {
	return m.DeleteFn(ctx, userID, mediaID)
}
func (m *MockMediaRepository) WithQuerier(_ database.Querier) repository.MediaRepository {
	return m
}
