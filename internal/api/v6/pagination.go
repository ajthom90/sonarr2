package v6

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Page is a generic cursor-paginated response envelope.
type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination holds the cursor pagination metadata.
type Pagination struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// cursorPayload is the JSON object encoded inside a cursor string.
type cursorPayload struct {
	LastID int64 `json:"lastId"`
}

// EncodeCursor creates a base64url-encoded cursor from a lastId value.
func EncodeCursor(lastID int64) string {
	payload, _ := json.Marshal(cursorPayload{LastID: lastID})
	return base64.URLEncoding.EncodeToString(payload)
}

// DecodeCursor extracts lastId from a base64url-encoded cursor string.
func DecodeCursor(cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, fmt.Errorf("invalid cursor payload: %w", err)
	}
	return payload.LastID, nil
}

// ParsePaginationParams reads limit and cursor from query parameters.
// Defaults: limit=50, lastID=0 (start from beginning).
// Max limit is 500.
func ParsePaginationParams(r *http.Request) (limit int, lastID int64, err error) {
	q := r.URL.Query()

	limit = 50
	if lStr := q.Get("limit"); lStr != "" {
		l, parseErr := strconv.Atoi(lStr)
		if parseErr != nil || l <= 0 {
			return 0, 0, fmt.Errorf("invalid limit: must be a positive integer")
		}
		if l > 500 {
			l = 500
		}
		limit = l
	}

	if cursor := q.Get("cursor"); cursor != "" {
		lastID, err = DecodeCursor(cursor)
		if err != nil {
			return 0, 0, err
		}
	}

	return limit, lastID, nil
}
