// Package client provides an HTTP client for communicating with the go-ls server.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vitorecomp/go-ls/pkg/models"
)

// Client communicates with a remote go-ls server over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a new Client targeting the given server baseURL (e.g. "http://localhost:8080").
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Health checks if the server's /healthz endpoint responds with 200 OK.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected health status: %s", resp.Status)
	}
	return nil
}

// UploadScan sends a completed ScanRecord to the server and returns the persisted ScanSummary.
func (c *Client) UploadScan(ctx context.Context, scan models.ScanRecord) (*models.ScanSummary, error) {
	body, err := json.Marshal(scan)
	if err != nil {
		return nil, fmt.Errorf("marshaling scan record: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/scans", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var summary models.ScanSummary
	if err := c.doJSON(req, http.StatusCreated, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// ListComputers retrieves all tracked computers and their directory snapshots from the server.
func (c *Client) ListComputers(ctx context.Context) ([]models.Computer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/computers", nil)
	if err != nil {
		return nil, err
	}

	var computers []models.Computer
	if err := c.doJSON(req, http.StatusOK, &computers); err != nil {
		return nil, err
	}
	return computers, nil
}

// ListScans retrieves summaries of all scans stored on the server.
func (c *Client) ListScans(ctx context.Context) ([]models.ScanSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/scans", nil)
	if err != nil {
		return nil, err
	}

	var summaries []models.ScanSummary
	if err := c.doJSON(req, http.StatusOK, &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

// GetScan retrieves a stored ScanRecord (including all entries) by scan ID.
func (c *Client) GetScan(ctx context.Context, scanID string) (*models.ScanRecord, error) {
	endpoint := fmt.Sprintf("%s/api/v1/scans/%s", c.baseURL, url.PathEscape(scanID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var rec models.ScanRecord
	if err := c.doJSON(req, http.StatusOK, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetScanHistory retrieves the historical revisions of a directory snapshot by scan ID.
func (c *Client) GetScanHistory(ctx context.Context, scanID string) ([]models.ScanHistoryEntry, error) {
	endpoint := fmt.Sprintf("%s/api/v1/scans/%s/history", c.baseURL, url.PathEscape(scanID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var history []models.ScanHistoryEntry
	if err := c.doJSON(req, http.StatusOK, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// DiffScans requests a comparison report between baseID and targetID from the server.
func (c *Client) DiffScans(ctx context.Context, baseID, targetID string) (*models.DiffReport, error) {
	params := url.Values{}
	params.Set("base", baseID)
	params.Set("target", targetID)
	endpoint := fmt.Sprintf("%s/api/v1/diff?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var report models.DiffReport
	if err := c.doJSON(req, http.StatusOK, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// FindDuplicates requests duplicate file groups within a single scan (if scanID is non-empty)
// or across all stored scans on the server.
func (c *Client) FindDuplicates(ctx context.Context, scanID string) ([]models.DuplicateGroup, error) {
	endpoint := c.baseURL + "/api/v1/duplicates"
	if scanID != "" {
		params := url.Values{}
		params.Set("scan_id", scanID)
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var groups []models.DuplicateGroup
	if err := c.doJSON(req, http.StatusOK, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *Client) doJSON(req *http.Request, expectedStatus int, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", req.URL.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var apiErr map[string]string
		if json.Unmarshal(raw, &apiErr) == nil && apiErr["error"] != "" {
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, apiErr["error"])
		}
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response JSON: %w", err)
	}
	return nil
}
