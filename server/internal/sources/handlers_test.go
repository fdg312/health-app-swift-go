package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fdg312/health-hub/internal/checkins"
	"github.com/fdg312/health-hub/internal/storage/memory"
)

func TestSourcesHandlers(t *testing.T) {
	// Setup
	memStorage := memory.New()
	sourcesStorage := memStorage.GetSourcesStorage()
	service := NewService(sourcesStorage, memStorage, nil, 10, "image/jpeg,image/png", 4, "", false)
	handlers := NewHandlers(service)

	// Get owner profile
	profiles, _ := memStorage.ListProfiles(context.Background())
	ownerID := profiles[0].ID

	t.Run("CreateLinkSource", func(t *testing.T) {
		url := "https://example.com"
		title := "Test Link"
		req := CreateSourceRequest{
			ProfileID: ownerID,
			Kind:      KindLink,
			Title:     &title,
			URL:       &url,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.HandleCreate(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var dto SourceDTO
		json.NewDecoder(w.Body).Decode(&dto)

		if dto.Kind != KindLink {
			t.Errorf("Expected kind link, got %s", dto.Kind)
		}
	})

	t.Run("CreateNoteSource", func(t *testing.T) {
		text := "Test note content"
		req := CreateSourceRequest{
			ProfileID: ownerID,
			Kind:      KindNote,
			Text:      &text,
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.HandleCreate(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("UploadImageSource", func(t *testing.T) {
		// Create multipart form
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		writer.WriteField("profile_id", ownerID.String())
		writer.WriteField("title", "Test Image")

		// Add fake image file with proper MIME type
		h := make(map[string][]string)
		h["Content-Type"] = []string{"image/png"}
		part, _ := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="file"; filename="test.png"`},
			"Content-Type":        {"image/png"},
		})
		part.Write([]byte("fake png data"))

		writer.Close()

		httpReq := httptest.NewRequest("POST", "/v1/sources/image", &buf)
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handlers.HandleCreateImage(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var dto SourceDTO
		json.NewDecoder(w.Body).Decode(&dto)

		if dto.Kind != KindImage {
			t.Errorf("Expected kind image, got %s", dto.Kind)
		}
	})

	t.Run("ListSourcesByCheckin", func(t *testing.T) {
		// Create checkin
		checkin := &checkins.Checkin{
			ProfileID: ownerID,
			Date:      "2026-02-13",
			Type:      "morning",
			Score:     4,
		}
		memStorage.GetCheckinsStorage().UpsertCheckin(checkin)

		// Upload image for checkin
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("profile_id", ownerID.String())
		writer.WriteField("checkin_id", checkin.ID.String())
		part, _ := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="file"; filename="test.jpg"`},
			"Content-Type":        {"image/jpeg"},
		})
		part.Write([]byte("fake jpg data"))
		writer.Close()

		httpReq := httptest.NewRequest("POST", "/v1/sources/image", &buf)
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		handlers.HandleCreateImage(w, httpReq)

		// List sources for checkin
		httpReq = httptest.NewRequest("GET", "/v1/sources?profile_id="+ownerID.String()+"&checkin_id="+checkin.ID.String(), nil)
		w = httptest.NewRecorder()
		handlers.HandleList(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp SourcesResponse
		json.NewDecoder(w.Body).Decode(&resp)

		if len(resp.Sources) == 0 {
			t.Errorf("Expected at least 1 source for checkin")
		}
	})

	t.Run("DownloadImage", func(t *testing.T) {
		// Upload image
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("profile_id", ownerID.String())
		part, _ := writer.CreatePart(map[string][]string{
			"Content-Disposition": {`form-data; name="file"; filename="download.png"`},
			"Content-Type":        {"image/png"},
		})
		testData := []byte("test image data for download")
		part.Write(testData)
		writer.Close()

		httpReq := httptest.NewRequest("POST", "/v1/sources/image", &buf)
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		handlers.HandleCreateImage(w, httpReq)

		var dto SourceDTO
		json.NewDecoder(w.Body).Decode(&dto)

		// Download it
		httpReq = httptest.NewRequest("GET", "/v1/sources/"+dto.ID.String()+"/download", nil)
		httpReq.SetPathValue("id", dto.ID.String())
		w = httptest.NewRecorder()
		handlers.HandleDownload(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "image/png" {
			t.Logf("Content-Type: %s (expected image/png)", contentType)
		}

		downloaded, _ := io.ReadAll(w.Body)
		if !bytes.Equal(downloaded, testData) {
			t.Errorf("Downloaded data mismatch")
		}
	})

	t.Run("DeleteSource", func(t *testing.T) {
		// Create source
		url := "https://delete.example.com"
		req := CreateSourceRequest{
			ProfileID: ownerID,
			Kind:      KindLink,
			URL:       &url,
		}
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/v1/sources", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handlers.HandleCreate(w, httpReq)

		var dto SourceDTO
		json.NewDecoder(w.Body).Decode(&dto)

		// Delete it
		httpReq = httptest.NewRequest("DELETE", "/v1/sources/"+dto.ID.String(), nil)
		httpReq.SetPathValue("id", dto.ID.String())
		w = httptest.NewRecorder()
		handlers.HandleDelete(w, httpReq)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify deleted
		httpReq = httptest.NewRequest("GET", "/v1/sources/"+dto.ID.String()+"/download", nil)
		httpReq.SetPathValue("id", dto.ID.String())
		w = httptest.NewRecorder()
		handlers.HandleDownload(w, httpReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 after delete, got %d", w.Code)
		}
	})

	t.Run("MaxSourcesPerCheckin", func(t *testing.T) {
		// Create checkin
		checkin := &checkins.Checkin{
			ProfileID: ownerID,
			Date:      "2026-02-15",
			Type:      "morning",
			Score:     3,
		}
		memStorage.GetCheckinsStorage().UpsertCheckin(checkin)

		// Upload 4 sources (max allowed)
		successCount := 0
		for i := 0; i < 5; i++ {
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)
			writer.WriteField("profile_id", ownerID.String())
			writer.WriteField("checkin_id", checkin.ID.String())
			part, _ := writer.CreatePart(map[string][]string{
				"Content-Disposition": {`form-data; name="file"; filename="test.png"`},
				"Content-Type":        {"image/png"},
			})
			part.Write([]byte("fake data"))
			writer.Close()

			httpReq := httptest.NewRequest("POST", "/v1/sources/image", &buf)
			httpReq.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()
			handlers.HandleCreateImage(w, httpReq)

			if w.Code == http.StatusCreated {
				successCount++
			}
		}

		// Should enforce max limit (allow at most 4)
		if successCount > 4 {
			t.Errorf("Expected at most 4 successful uploads, got %d", successCount)
		}
		if successCount < 3 {
			t.Errorf("Expected at least 3 successful uploads, got %d", successCount)
		}
	})
}
