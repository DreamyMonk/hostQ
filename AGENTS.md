# hostQ Agent Notes

hostQ is now a single native control panel.

- Production entrypoint: `main.go`
- Runtime: one `hostq-panel` systemd service behind Nginx
- Validation: `go test ./...` and `go build .`
- Do not reintroduce another web runtime for the panel.
