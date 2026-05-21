package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	w io.Writer
}

func (g *gzipWriter) Write(p []byte) (int, error) { return g.w.Write(p) }

// gzipMiddleware compresses responses when the client advertises gzip support.
// Nginx will usually do this for us in production, but when the panel is hit
// directly on :8090 (initial setup) this still saves a meaningful amount of
// bytes on the inline-everything HTML.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		next.ServeHTTP(&gzipWriter{ResponseWriter: w, w: gz}, r)
	})
}
