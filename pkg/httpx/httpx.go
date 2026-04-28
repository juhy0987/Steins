// Package httpx provides small HTTP helpers — JSON encode/decode and
// error response writing — used by the API handlers.
//
// httpx 패키지는 API 핸들러가 사용하는 JSON 인코딩/디코딩 및 에러 응답
// 헬퍼를 제공합니다.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"steins/internal/apperr"
)

// WriteJSON writes a JSON response with the given status and payload.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteData wraps the payload in `{ "data": ... }` and writes it.
func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, map[string]any{"data": data})
}

// WriteList wraps a list payload with an optional `meta` block.
func WriteList(w http.ResponseWriter, status int, data any, meta map[string]any) {
	body := map[string]any{"data": data}
	if meta != nil {
		body["meta"] = meta
	}
	WriteJSON(w, status, body)
}

// WriteError translates an error into a JSON error response.
// If err is not an *apperr.Error it is treated as an internal error.
func WriteError(w http.ResponseWriter, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		ae = apperr.NewInternalError("unexpected error", err)
	}

	body := map[string]any{
		"error": map[string]any{
			"code":    ae.Code,
			"message": ae.Message,
		},
	}
	if ae.Details != nil {
		body["error"].(map[string]any)["details"] = ae.Details
	}

	WriteJSON(w, ae.HTTPStatus(), body)
}

// DecodeJSON decodes the request body into the given target.
func DecodeJSON(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}
	return nil
}
