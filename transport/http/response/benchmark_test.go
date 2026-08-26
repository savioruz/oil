package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkWriteJSON measures JSON encoding of a typical success payload.
// Used to compare encoding/json v1 vs v2 encode cost across Go versions.
func BenchmarkWriteJSON(b *testing.B) {
	payload := Data[any]{Data: map[string]any{
		"id":          "01J2V4XKQ7R8M9N0P1Q2R3S4T",
		"title":       "A moderately sized title for the benchmark payload",
		"description": "A longer description string that exercises the JSON encoder with enough bytes to be representative of real handler responses.",
		"completed":   true,
		"count":       42,
		"tags":        []string{"one", "two", "three"},
	}}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, payload)
	}
}
