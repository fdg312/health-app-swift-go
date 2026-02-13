package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const (
	defaultAPIBase = "http://localhost:8080"
)

var (
	apiBase    string
	token      string
	profileID  string
	client     = &http.Client{Timeout: 30 * time.Second}
	testDate   string
	createdIDs = make(map[string]string) // track created resources for cleanup
)

func main() {
	fmt.Println("=== Health Hub E2E Smoke Test ===")
	fmt.Println()

	// Load config from env
	apiBase = getEnv("API_BASE_URL", defaultAPIBase)
	token = getEnv("SMOKE_TOKEN", "")
	profileID = getEnv("SMOKE_PROFILE_ID", "")

	fmt.Printf("API Base: %s\n", apiBase)
	fmt.Printf("Token: %s\n", maskString(token))
	fmt.Printf("Profile ID: %s\n", maskString(profileID))
	fmt.Println()

	// Test date (today)
	testDate = time.Now().Format("2006-01-02")

	// Run smoke tests
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Healthz", testHealthz},
		{"Get Profile ID", testGetProfileID},
		{"Sync Batch", testSyncBatch},
		{"Create Checkin", testCreateCheckin},
		{"Get Feed Day", testGetFeedDay},
		{"Create Report (CSV)", testCreateReportPDF},
		{"List Reports", testListReports},
		{"Download Report", testDownloadReport},
		{"Upload Source (Image)", testUploadSourceImage},
		{"List Sources", testListSources},
		{"Download Source", testDownloadSource},
		{"Delete Source", testDeleteSource},
		{"Delete Report", testDeleteReport},
	}

	failed := false
	for i, step := range steps {
		fmt.Printf("[%d/%d] %s... ", i+1, len(steps), step.name)
		if err := step.fn(); err != nil {
			fmt.Printf("❌ FAILED\n")
			fmt.Printf("  Error: %v\n\n", err)
			failed = true
			break
		}
		fmt.Printf("✅ OK\n")
	}

	fmt.Println()
	if failed {
		fmt.Println("❌ SMOKE TEST FAILED")
		os.Exit(1)
	}

	fmt.Println("✅ ALL SMOKE TESTS PASSED")
}

func testHealthz() error {
	req, err := http.NewRequest("GET", apiBase+"/healthz", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func testGetProfileID() error {
	// If profile ID already set via env, skip
	if profileID != "" {
		return nil
	}

	req, err := http.NewRequest("GET", apiBase+"/v1/profiles", nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Profiles []struct {
			ID        string `json:"id"`
			IsOwner   bool   `json:"is_owner"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"profiles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	if len(result.Profiles) == 0 {
		return fmt.Errorf("no profiles found")
	}

	// Find owner profile
	for _, p := range result.Profiles {
		if p.IsOwner {
			profileID = p.ID
			return nil
		}
	}

	// Fallback to first profile
	profileID = result.Profiles[0].ID
	return nil
}

func testSyncBatch() error {
	payload := map[string]interface{}{
		"profile_id": profileID,
		"daily": []map[string]interface{}{
			{
				"date":        testDate,
				"steps":       8000,
				"sleep_hours": 7.5,
				"water_ml":    2000,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiBase+"/v1/sync/batch", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func testCreateCheckin() error {
	payload := map[string]interface{}{
		"profile_id": profileID,
		"date":       testDate,
		"type":       "morning",
		"score":      4,
		"note":       "Smoke test checkin",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiBase+"/v1/checkins", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept both 200 (upsert existing) and 201 (created new)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	createdIDs["checkin"] = result.ID
	return nil
}

func testGetFeedDay() error {
	url := fmt.Sprintf("%s/v1/feed/day?profile_id=%s&date=%s", apiBase, profileID, testDate)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func testCreateReportPDF() error {
	fromDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	toDate := testDate

	payload := map[string]interface{}{
		"profile_id": profileID,
		"format":     "csv",
		"from":       fromDate,
		"to":         toDate,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiBase+"/v1/reports", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		ID        string `json:"id"`
		SizeBytes int64  `json:"size_bytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	if result.SizeBytes < 10 {
		return fmt.Errorf("report size is %d bytes (too small)", result.SizeBytes)
	}

	createdIDs["report"] = result.ID
	return nil
}

func testListReports() error {
	url := fmt.Sprintf("%s/v1/reports?profile_id=%s", apiBase, profileID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Reports []struct {
			ID string `json:"id"`
		} `json:"reports"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	if len(result.Reports) == 0 {
		return fmt.Errorf("no reports found")
	}

	return nil
}

func testDownloadReport() error {
	reportID := createdIDs["report"]
	if reportID == "" {
		return fmt.Errorf("no report ID to download")
	}

	url := fmt.Sprintf("%s/v1/reports/%s/download", apiBase, reportID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	// Don't follow redirects automatically - we need to check redirect behavior
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() { client.CheckRedirect = originalCheckRedirect }()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept 200 (direct serve) or 302 (redirect)
	if resp.StatusCode == http.StatusOK {
		// Direct serve (local mode)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read body: %w", err)
		}
		if len(data) < 10 {
			return fmt.Errorf("report too small: %d bytes", len(data))
		}
		return nil
	}

	if resp.StatusCode == http.StatusFound {
		// Redirect (S3 mode)
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("redirect without Location header")
		}

		// Follow redirect
		getReq, err := http.NewRequest("GET", location, nil)
		if err != nil {
			return fmt.Errorf("failed to create redirect request: %w", err)
		}

		getResp, err := client.Do(getReq)
		if err != nil {
			return fmt.Errorf("failed to follow redirect: %w", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(getResp.Body, 4096))
			return fmt.Errorf("redirect failed: status=%d body=%s", getResp.StatusCode, string(body))
		}

		data, err := io.ReadAll(getResp.Body)
		if err != nil {
			return fmt.Errorf("failed to read redirected body: %w", err)
		}
		if len(data) < 10 {
			return fmt.Errorf("report too small: %d bytes", len(data))
		}
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("unexpected status=%d body=%s", resp.StatusCode, string(body))
}

func testUploadSourceImage() error {
	// Generate a minimal PNG image (1x1 pixel, red)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // width=1, height=1
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Create multipart body
	var b bytes.Buffer
	boundary := "----SmokeTestBoundary123"
	w := io.Writer(&b)

	// Write profile_id field
	fmt.Fprintf(w, "--%s\r\n", boundary)
	fmt.Fprintf(w, "Content-Disposition: form-data; name=\"profile_id\"\r\n\r\n")
	fmt.Fprintf(w, "%s\r\n", profileID)

	// Write title field
	fmt.Fprintf(w, "--%s\r\n", boundary)
	fmt.Fprintf(w, "Content-Disposition: form-data; name=\"title\"\r\n\r\n")
	fmt.Fprintf(w, "Smoke Test Image\r\n")

	// Write file field
	fmt.Fprintf(w, "--%s\r\n", boundary)
	fmt.Fprintf(w, "Content-Disposition: form-data; name=\"file\"; filename=\"test.png\"\r\n")
	fmt.Fprintf(w, "Content-Type: image/png\r\n\r\n")
	w.Write(pngData)
	fmt.Fprintf(w, "\r\n--%s--\r\n", boundary)

	req, err := http.NewRequest("POST", apiBase+"/v1/sources/image", &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		ID        string `json:"id"`
		SizeBytes int64  `json:"size_bytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	if result.SizeBytes <= 0 {
		return fmt.Errorf("source size is %d bytes", result.SizeBytes)
	}

	createdIDs["source"] = result.ID
	return nil
}

func testListSources() error {
	url := fmt.Sprintf("%s/v1/sources?profile_id=%s", apiBase, profileID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Sources []struct {
			ID string `json:"id"`
		} `json:"sources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	if len(result.Sources) == 0 {
		return fmt.Errorf("no sources found")
	}

	return nil
}

func testDownloadSource() error {
	sourceID := createdIDs["source"]
	if sourceID == "" {
		return fmt.Errorf("no source ID to download")
	}

	url := fmt.Sprintf("%s/v1/sources/%s/download", apiBase, sourceID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	// Don't follow redirects automatically
	originalCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() { client.CheckRedirect = originalCheckRedirect }()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Accept 200 (direct serve) or 302 (redirect)
	if resp.StatusCode == http.StatusOK {
		// Direct serve (local mode)
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read body: %w", err)
		}
		if len(data) < 10 {
			return fmt.Errorf("source too small: %d bytes", len(data))
		}
		return nil
	}

	if resp.StatusCode == http.StatusFound {
		// Redirect (S3 mode)
		location := resp.Header.Get("Location")
		if location == "" {
			return fmt.Errorf("redirect without Location header")
		}

		// Follow redirect
		getReq, err := http.NewRequest("GET", location, nil)
		if err != nil {
			return fmt.Errorf("failed to create redirect request: %w", err)
		}

		getResp, err := client.Do(getReq)
		if err != nil {
			return fmt.Errorf("failed to follow redirect: %w", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(getResp.Body, 4096))
			return fmt.Errorf("redirect failed: status=%d body=%s", getResp.StatusCode, string(body))
		}

		data, err := io.ReadAll(getResp.Body)
		if err != nil {
			return fmt.Errorf("failed to read redirected body: %w", err)
		}
		if len(data) < 10 {
			return fmt.Errorf("source too small: %d bytes", len(data))
		}
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("unexpected status=%d body=%s", resp.StatusCode, string(body))
}

func testDeleteSource() error {
	sourceID := createdIDs["source"]
	if sourceID == "" {
		return fmt.Errorf("no source ID to delete")
	}

	url := fmt.Sprintf("%s/v1/sources/%s", apiBase, sourceID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func testDeleteReport() error {
	reportID := createdIDs["report"]
	if reportID == "" {
		return fmt.Errorf("no report ID to delete")
	}

	url := fmt.Sprintf("%s/v1/reports/%s", apiBase, reportID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	addAuth(req)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper functions

func addAuth(req *http.Request) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func maskString(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
