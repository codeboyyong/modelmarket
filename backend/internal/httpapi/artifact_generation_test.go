package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGeminiImageSavedBeforePreview(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("GEMINI_API_KEY", "test-key")
	store := &fakeObjectStore{}
	app.ObjectStore = store
	app.Config.ObjectStorageProvider = "s3"
	image := []byte("test image bytes")
	app.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload struct {
			GenerationConfig struct {
				ResponseModalities []string `json:"responseModalities"`
			} `json:"generationConfig"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if strings.Join(payload.GenerationConfig.ResponseModalities, ",") != "TEXT,IMAGE" {
			t.Fatal("image output not requested")
		}
		body := `{"candidates":[{"content":{"parts":[{"thought":true,"text":"private thought"},{"inlineData":{"mimeType":"image/png","data":"` + base64.StdEncoding.EncodeToString(image) + `"}}]}}],"usageMetadata":{"candidatesTokenCount":12}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	result, err := app.callGeminiGenerateContent(context.Background(), selectedModelRoute{UpstreamModelID: "gemini-test-image", ModelModality: "image"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Content, "private thought") {
		t.Fatal("thought leaked")
	}
	mock.ExpectExec("insert into user_workbench_assets").WillReturnResult(sqlmock.NewResult(1, 1))
	content, assets := app.rehostExternalFilesInContent(context.Background(), "project", "", "", "request", result.Content)
	if len(assets) != 1 || !strings.Contains(content, "/api/v1/assets/") || strings.Contains(content, "base64") || string(store.putBody) != string(image) {
		t.Fatalf("artifact not stored: %s", content)
	}
	// Both preview and download resolve through our endpoint to the stored bytes.
	mock.ExpectQuery("select coalesce\\(object_key").WillReturnRows(sqlmock.NewRows([]string{"object_key", "mime_type"}).AddRow(store.putKey, "image/png"))
	req := httptest.NewRequest("GET", assets[0]["download_url"].(string), nil)
	req.SetPathValue("id", assets[0]["id"].(string))
	req.SetPathValue("filename", "image.png")
	rec := httptest.NewRecorder()
	app.downloadAsset(rec, req)
	if rec.Code != 200 || rec.Body.String() != string(image) {
		t.Fatalf("download failed: %d %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactImportFailuresAreVisible(t *testing.T) {
	for _, mode := range []string{"fetch", "storage"} {
		t.Run(mode, func(t *testing.T) {
			app, mock, cleanup := testApp(t)
			defer cleanup()
			app.Config.ObjectStorageProvider = "s3" // Deliberately no initialized store.
			content := "![Portrait](data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("image bytes")) + ")"
			if mode == "fetch" {
				content = "![Portrait](https://example.test/missing.png)"
				app.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("missing"))}, nil
				})}
			}
			got, assets := app.rehostExternalFilesInContent(context.Background(), "p", "", "", "req", content)
			if len(assets) != 0 || !strings.HasPrefix(got, "Artifact unavailable:") || strings.Contains(got, "![") {
				t.Fatalf("failure hidden: %s", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGeminiRejectsIncompleteImageGeneration(t *testing.T) {
	for _, parts := range []string{`[{"text":"Unable to generate"}]`, `[{"inlineData":{"mimeType":"image/png","data":"invalid base64"}}]`} {
		app, _, cleanup := testApp(t)
		t.Setenv("GEMINI_API_KEY", "test-key")
		app.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":` + parts + `}}]}`)), Header: make(http.Header)}, nil
		})}
		_, err := app.callGeminiGenerateContent(context.Background(), selectedModelRoute{ModelModality: "image", UpstreamModelID: "test-image"}, nil)
		cleanup()
		if err == nil {
			t.Fatal("incomplete generation accepted")
		}
	}
}
