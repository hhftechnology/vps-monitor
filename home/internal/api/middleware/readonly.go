package middleware

import (
	"encoding/json"
	"net/http"
)

// ReadOnly creates a middleware that blocks mutating requests when in read-only mode.
// The isReadOnly function is evaluated per request, supporting hot-reload.
func ReadOnly(isReadOnly func() bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isReadOnly() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)

				response := map[string]any{
					"error":     "Operation not allowed in read-only mode",
					"read_only": true,
				}

				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
