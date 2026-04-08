package httpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/clementhaon/sandbox-api-go/pkg/models"
)

// UserServiceClient provides methods to call the User Service API.
type UserServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewUserServiceClient creates a new client for the User Service.
func NewUserServiceClient(baseURL string) *UserServiceClient {
	return &UserServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetUserBrief fetches a brief user profile from the User Service.
func (c *UserServiceClient) GetUserBrief(userID int) (*models.UserBrief, error) {
	url := fmt.Sprintf("%s/internal/users/%d/brief", c.baseURL, userID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call user service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status %d", resp.StatusCode)
	}

	var user models.UserBrief
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}
