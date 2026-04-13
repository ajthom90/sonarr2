package v6

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteErrorRFC9457 verifies that WriteError produces a response with
// Content-Type application/problem+json and the correct RFC 9457 JSON shape.
func TestWriteErrorRFC9457(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v6/series/99", nil)

	WriteError(rr, req, http.StatusNotFound, "No series with id 99")

	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}

	var pd ProblemDetail
	if err := json.NewDecoder(rr.Body).Decode(&pd); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pd.Type != "about:blank" {
		t.Errorf("type = %q, want about:blank", pd.Type)
	}
	if pd.Title != "Not Found" {
		t.Errorf("title = %q, want Not Found", pd.Title)
	}
	if pd.Status != 404 {
		t.Errorf("status = %d, want 404", pd.Status)
	}
	if pd.Detail != "No series with id 99" {
		t.Errorf("detail = %q, want 'No series with id 99'", pd.Detail)
	}
	if pd.Instance != "/api/v6/series/99" {
		t.Errorf("instance = %q, want /api/v6/series/99", pd.Instance)
	}
}

// TestCursorRoundtrip verifies that encoding then decoding a cursor returns
// the same lastID.
func TestCursorRoundtrip(t *testing.T) {
	tests := []int64{0, 1, 42, 999999}
	for _, id := range tests {
		cursor := EncodeCursor(id)
		got, err := DecodeCursor(cursor)
		if err != nil {
			t.Errorf("DecodeCursor(%q): %v", cursor, err)
			continue
		}
		if got != id {
			t.Errorf("roundtrip id=%d: got %d", id, got)
		}
	}
}

// TestParsePaginationDefaults verifies that a request with no query params
// returns the default limit of 50 and a lastID of 0.
func TestParsePaginationDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v6/series", nil)
	limit, lastID, err := ParsePaginationParams(req)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if limit != 50 {
		t.Errorf("limit = %d, want 50", limit)
	}
	if lastID != 0 {
		t.Errorf("lastID = %d, want 0", lastID)
	}
}

// TestParsePaginationCustomLimit verifies that a custom limit is respected.
func TestParsePaginationCustomLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v6/series?limit=100", nil)
	limit, lastID, err := ParsePaginationParams(req)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if limit != 100 {
		t.Errorf("limit = %d, want 100", limit)
	}
	if lastID != 0 {
		t.Errorf("lastID = %d, want 0", lastID)
	}
}

// TestParsePaginationMaxLimit verifies that limits above 500 are capped.
func TestParsePaginationMaxLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v6/series?limit=9999", nil)
	limit, _, err := ParsePaginationParams(req)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if limit != 500 {
		t.Errorf("limit = %d, want 500 (capped)", limit)
	}
}

// TestParsePaginationWithCursor verifies that a cursor is decoded correctly.
func TestParsePaginationWithCursor(t *testing.T) {
	cursor := EncodeCursor(42)
	req := httptest.NewRequest(http.MethodGet, "/api/v6/series?cursor="+cursor, nil)
	_, lastID, err := ParsePaginationParams(req)
	if err != nil {
		t.Fatalf("ParsePaginationParams: %v", err)
	}
	if lastID != 42 {
		t.Errorf("lastID = %d, want 42", lastID)
	}
}
