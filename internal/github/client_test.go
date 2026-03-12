package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// mockBackend implements Backend for unit testing the Client wrapper.
type mockBackend struct {
	event   map[string]interface{}
	pr      PullRequest
	reviews []Review
	err     error
}

func (m *mockBackend) ReadEvent() (map[string]interface{}, error) {
	return m.event, m.err
}

func (m *mockBackend) GetPR(prNumber int) (PullRequest, error) {
	return m.pr, m.err
}

func (m *mockBackend) GetPRReviews(prNumber int) ([]Review, error) {
	return m.reviews, m.err
}

func TestClient_ReadEvent_delegates_to_backend(t *testing.T) {
	expected := map[string]interface{}{"action": "submitted"}
	mock := &mockBackend{event: expected}
	client := NewClient(mock)

	event, err := client.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event["action"] != "submitted" {
		t.Errorf("expected action=submitted, got %v", event["action"])
	}
}

func TestClient_GetPR_delegates_to_backend(t *testing.T) {
	expected := PullRequest{State: "open", Merged: false, MergeableState: "clean"}
	mock := &mockBackend{pr: expected}
	client := NewClient(mock)

	pr, err := client.GetPR(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr != expected {
		t.Errorf("expected %+v, got %+v", expected, pr)
	}
}

func TestClient_GetPRReviews_delegates_to_backend(t *testing.T) {
	expected := []Review{{State: "approved", Username: "alice"}}
	mock := &mockBackend{reviews: expected}
	client := NewClient(mock)

	reviews, err := client.GetPRReviews(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reviews) != 1 || reviews[0] != expected[0] {
		t.Errorf("expected %+v, got %+v", expected, reviews)
	}
}

// --- WebBackend tests ---

func TestWebBackend_ReadEvent_parses_json_file(t *testing.T) {
	dir := t.TempDir()
	eventFile := filepath.Join(dir, "event.json")
	payload := map[string]interface{}{
		"action": "submitted",
		"number": float64(7),
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(eventFile, data, 0644); err != nil {
		t.Fatalf("writing event file: %v", err)
	}

	backend := NewWebBackend(eventFile, "owner/repo", "fake-token")
	event, err := backend.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event["action"] != "submitted" {
		t.Errorf("expected action=submitted, got %v", event["action"])
	}
	if event["number"] != float64(7) {
		t.Errorf("expected number=7, got %v", event["number"])
	}
}

func TestWebBackend_ReadEvent_returns_error_for_missing_file(t *testing.T) {
	backend := NewWebBackend("/nonexistent/path.json", "owner/repo", "fake-token")
	_, err := backend.ReadEvent()
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestWebBackend_ReadEvent_returns_error_for_invalid_json(t *testing.T) {
	dir := t.TempDir()
	eventFile := filepath.Join(dir, "event.json")
	if err := os.WriteFile(eventFile, []byte("not json"), 0644); err != nil {
		t.Fatalf("writing event file: %v", err)
	}

	backend := NewWebBackend(eventFile, "owner/repo", "fake-token")
	_, err := backend.ReadEvent()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestWebBackend_GetPR_returns_pull_request(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		resp := map[string]interface{}{
			"state":           "open",
			"merged":          false,
			"mergeable_state": "clean",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend := NewWebBackend("", "owner/repo", "test-token")
	backend.baseURL = server.URL

	pr, err := backend.GetPR(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.State != "open" {
		t.Errorf("expected state=open, got %s", pr.State)
	}
	if pr.Merged != false {
		t.Errorf("expected merged=false, got %v", pr.Merged)
	}
	if pr.MergeableState != "clean" {
		t.Errorf("expected mergeable_state=clean, got %s", pr.MergeableState)
	}
}

func TestWebBackend_GetPR_returns_error_on_non_2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	backend := NewWebBackend("", "owner/repo", "test-token")
	backend.baseURL = server.URL

	_, err := backend.GetPR(999)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestWebBackend_GetPRReviews_returns_reviews_with_lowercased_state(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/42/reviews" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := []map[string]interface{}{
			{
				"state": "APPROVED",
				"user":  map[string]string{"login": "alice"},
			},
			{
				"state": "CHANGES_REQUESTED",
				"user":  map[string]string{"login": "bob"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend := NewWebBackend("", "owner/repo", "test-token")
	backend.baseURL = server.URL

	reviews, err := backend.GetPRReviews(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(reviews))
	}
	if reviews[0].State != "approved" || reviews[0].Username != "alice" {
		t.Errorf("review[0]: expected {approved alice}, got {%s %s}", reviews[0].State, reviews[0].Username)
	}
	if reviews[1].State != "changes_requested" || reviews[1].Username != "bob" {
		t.Errorf("review[1]: expected {changes_requested bob}, got {%s %s}", reviews[1].State, reviews[1].Username)
	}
}

func TestWebBackend_GetPRReviews_returns_empty_slice_for_no_reviews(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	backend := NewWebBackend("", "owner/repo", "test-token")
	backend.baseURL = server.URL

	reviews, err := backend.GetPRReviews(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reviews) != 0 {
		t.Errorf("expected 0 reviews, got %d", len(reviews))
	}
}
