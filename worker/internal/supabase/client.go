package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a minimal REST client for Supabase's PostgREST API, using the
// service role key. We avoid a full SDK dependency since the worker only
// needs a handful of operations (update scan_jobs, read repos).
type Client struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

func New(baseURL, serviceKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) headers() http.Header {
	h := http.Header{}
	h.Set("apikey", c.serviceKey)
	h.Set("Authorization", "Bearer "+c.serviceKey)
	h.Set("Content-Type", "application/json")
	h.Set("Prefer", "return=representation")
	return h
}

// GetRepoByID fetches a single repo row by id.
func (c *Client) GetRepoByID(id string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/v1/repos?id=eq.%s&select=*", c.baseURL, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = c.headers()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("supabase GET repos failed (%d): %s", resp.StatusCode, string(body))
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("repo %s not found", id)
	}
	return rows[0], nil
}

// UpdateScanJob patches a scan_jobs row (status, error, etc).
func (c *Client) UpdateScanJob(jobID string, patch map[string]interface{}) error {
	url := fmt.Sprintf("%s/rest/v1/scan_jobs?id=eq.%s", c.baseURL, jobID)

	payload, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header = c.headers()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("supabase PATCH scan_jobs failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}
