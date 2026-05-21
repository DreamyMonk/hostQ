package main

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// gzipResponseWriter wraps http.ResponseWriter so we can:
//  1. Decide gzip-vs-passthrough at WriteHeader time based on status code,
//     Content-Type, Content-Disposition, and existing Content-Encoding.
//  2. Sniff Content-Type from the original (uncompressed) bytes on the first
//     Write when the handler didn't set one — otherwise Go's
//     http.DetectContentType reads the gzip magic and labels HTML responses
//     as application/x-gzip.
//  3. Stay completely out of the way for redirects, 304s, attachments, and
//     already-binary content so file downloads and 3xx responses pass
//     through unchanged.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
	bypass      bool // true => writes go straight to ResponseWriter, no gzip
	gzStarted   bool // true => we have committed bytes to gz; needs Close()
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true

	h := g.ResponseWriter.Header()
	ct := strings.ToLower(h.Get("Content-Type"))
	cd := strings.ToLower(h.Get("Content-Disposition"))
	ce := h.Get("Content-Encoding")

	noBody := code < 200 ||
		code == http.StatusNoContent ||
		code == http.StatusResetContent ||
		code == http.StatusNotModified ||
		(code >= 300 && code < 400)

	switch {
	case noBody,
		ce != "",
		strings.HasPrefix(cd, "attachment"),
		ct != "" && !gzippableType(ct):
		g.bypass = true
	default:
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		h.Del("Content-Length")
		g.gzStarted = true
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		// Sniff Content-Type from the *original* bytes before we commit to gzip.
		h := g.ResponseWriter.Header()
		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", http.DetectContentType(p))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.bypass {
		return g.ResponseWriter.Write(p)
	}
	return g.gz.Write(p)
}

func (g *gzipResponseWriter) Flush() {
	if g.gzStarted {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func gzippableType(ct string) bool {
	if ct == "" {
		return false
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	switch ct {
	case "application/json",
		"application/javascript",
		"application/xml",
		"application/xhtml+xml",
		"application/atom+xml",
		"application/rss+xml",
		"image/svg+xml":
		return true
	}
	return false
}

// gzipMiddleware compresses responses for clients that advertise gzip support
// — but only when the response is text-like *and* carries a body. Redirects,
// 304s, attachments, and binary downloads stream through untouched so they
// keep their original status codes and Content-Type headers.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzip.NewWriter(w)
		gw := &gzipResponseWriter{ResponseWriter: w, gz: gz}
		defer func() {
			if gw.gzStarted {
				_ = gz.Close()
			}
		}()
		next.ServeHTTP(gw, r)
	})
}
