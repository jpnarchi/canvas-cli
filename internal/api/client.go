package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"canvas-cli/internal/config"
)

type Client struct {
	BaseURL    string
	Config     *config.Config
	HTTPClient *http.Client
	PerPage    int
	Debug      bool
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		BaseURL: strings.TrimRight(cfg.APIURL, "/") + "/api/v1",
		Config:  cfg,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		PerPage: 50,
	}
}

func (c *Client) debugf(format string, args ...interface{}) {
	if c.Debug {
		fmt.Printf("  [debug] "+format+"\n", args...)
	}
}

// TestConnection verifies that the API token is valid by calling /users/self/profile.
func (c *Client) TestConnection() (string, error) {
	c.debugf("Testing API connection to %s", c.BaseURL)

	req, err := http.NewRequest("GET", c.BaseURL+"/users/self/profile", nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("connecting to Canvas: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	c.debugf("Response status: %d", resp.StatusCode)

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("invalid API token — generate a new one in Canvas → Account → Settings → New Access Token")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var profile struct {
		Name string `json:"name"`
	}
	json.Unmarshal(body, &profile)
	return profile.Name, nil
}

func (c *Client) request(method, endpoint string, body io.Reader, contentType string) ([]byte, error) {
	u := c.BaseURL + endpoint
	if !strings.Contains(u, "per_page=") {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += fmt.Sprintf("%sper_page=%d", sep, c.PerPage)
	}

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Config.APIToken)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized — your API token may be invalid or expired. Run: canvas-cli configure")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("forbidden — you don't have permission for this action")
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found — check the ID and try again")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (c *Client) GET(endpoint string) ([]byte, error) {
	return c.request("GET", endpoint, nil, "")
}

func (c *Client) POST(endpoint string, form url.Values) ([]byte, error) {
	return c.request("POST", endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

func (c *Client) PUT(endpoint string, form url.Values) ([]byte, error) {
	return c.request("PUT", endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

func (c *Client) DELETE(endpoint string) ([]byte, error) {
	return c.request("DELETE", endpoint, nil, "")
}

func (c *Client) GetJSON(endpoint string, target interface{}) error {
	data, err := c.GET(endpoint)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (c *Client) GetPaginated(endpoint string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	page := 1
	for {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
		}
		pageURL := fmt.Sprintf("%s%spage=%d", endpoint, sep, page)
		data, err := c.GET(pageURL)
		if err != nil {
			return nil, err
		}
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(items) < c.PerPage {
			break
		}
		page++
	}
	return all, nil
}
