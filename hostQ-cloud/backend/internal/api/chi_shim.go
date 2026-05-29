package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// chiParam centralises URL param reads so handlers don't import chi directly.
func chiParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}
