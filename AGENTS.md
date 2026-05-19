# hostQ Agent Notes

hostQ is now a Go-only control panel.

- Production entrypoint: `cmd/hostq-panel/main.go`
- Runtime: one `hostq-panel` systemd service behind Nginx
- Validation: `go test ./cmd/hostq-panel` and `go build ./cmd/hostq-panel`
- Do not reintroduce another web runtime for the panel.
