package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

type Feature struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Rules       map[string]interface{} `json:"rules"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ClientOptions struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func NewClient(opts ClientOptions) *Client {
	if opts.BaseURL == "" {
		opts.BaseURL = "http://localhost:8080"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: opts.BaseURL,
		apiKey:  opts.APIKey,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
	}
}

func (c *Client) IsFeatureEnabled(featureName string, userID string) (bool, error) {
	feature, err := c.GetFeature(featureName)
	if err != nil {
		return false, err
	}
	
	return feature.Enabled, nil
}

func (c *Client) GetFeature(featureName string) (*Feature, error) {
	url := fmt.Sprintf("%s/api/v1/features/%s", c.baseURL, featureName)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}
	
	var feature Feature
	if err := json.NewDecoder(resp.Body).Decode(&feature); err != nil {
		return nil, err
	}
	
	return &feature, nil
}

func (c *Client) CreateFeature(feature *Feature) error {
	url := fmt.Sprintf("%s/api/v1/features", c.baseURL)
	
	body, err := json.Marshal(feature)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}
	
	return nil
}