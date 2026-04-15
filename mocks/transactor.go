package mocks

import (
	"context"

	"github.com/clementhaon/sandbox-api-go/database"
)

// MockTransactor implements database.Transactor for testing.
// It executes the function directly without a real transaction,
// passing nil as the Querier (mocks ignore it via WithQuerier returning self).
type MockTransactor struct{}

func (m *MockTransactor) WithTransaction(_ context.Context, fn func(q database.Querier) error) error {
	return fn(nil)
}
