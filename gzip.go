package main

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipResponseWriter wraps http.ResponseWriter so we can:
//  1. Sniff Content-Type from the original (uncompressed) bytes, not the
//     gzipped ones — otherwise Go's http.DetectContentType sees the gzip
//     magic bytes (1f 8b 08) and tags the response application/x-gzip,
//     which the browser then refuses to render as HTML.
//  2. Decide *not* to gzip non-text payloads (file downloads, zips, images).
type gzipResponseWriter struct {
	http.ResponseWriter
	gz             *gzip.Writer
	wroteHeader    bool
	bypass         bool // true => stream straight through, no gzip
	statusBuffered int
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.statusBuffered = code
	// don't flush headers yet; first Write decides bypass + sets CE
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		g.wroteHeader = true
		h := g.ResponseWriter.Header()

		// Sniff from the *original* bytes when no Content-Type is set.
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", http.DetectContentType(p))
		}
		ct := strings.ToLower(h.Get("Content-Type"))
		cd := strings.ToLower(h.Get("Content-Disposition"))
		ce := h.Get("Content-Encoding")

		// Bypass when the response is binary, already compressed, or an
		// attachment download — gzip there is wasteful and risks double-encoding.
		if ce != "" || strings.HasPrefix(cd, "attachment") || !gzippableType(ct) {
			g.bypass = true
			h.Del("Content-Encoding") // make sure we didn't pre-set this
		} else {
			h.Set("Content-Encoding", "gzip")
			h.Add("Vary", "Accept-Encoding")
			h.Del("Content-Length")
		}
		status := g.statusBuffered
		if status == 0 {
			status = http.StatusOK
		}
		g.ResponseWriter.WriteHeader(status)
	}
	if g.bypass {
		return g.ResponseWriter.Write(p)
	}
	return g.gz.Write(p)
}

func (g *gzipResponseWriter) Flush() {
	if g.gz != nil && !g.bypass {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func gzippableType(ct string) bool {
	ct = strings.ToLower(ct)
	if ct == "" {
		return false
	}
	// Strip parameters like "; charset=utf-8".
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	for _, suffix := range []string{
		"application/json",
		"application/javascript",
		"application/xml",
		"application/xhtml+xml",
		"application/atom+xml",
		"application/rss+xml",
		"image/svg+xml",
	} {
		if ct == suffix {
			return true
		}
	}
	return false
}

// gzipMiddleware compresses responses for clients that advertise gzip support
// — but only when the response is text-like. File downloads, zips, and images
// are streamed through untouched.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzip.NewWriter(w)
		gw := &gzipResponseWriter{ResponseWriter: w, gz: gz}
		defer func() {
			if !gw.bypass {
				_ = gz.Close()
			}
		}()
		next.ServeHTTP(gw, r)
	})
}
