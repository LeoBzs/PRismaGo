// Package httpx holds small JSON request/response helpers shared by handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(v)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Decode reads a JSON body into dst, rejecting unknown fields and trailing data.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if dec.More() {
		return errors.New("invalid JSON body: multiple values")
	}
	return nil
}
