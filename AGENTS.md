# AGENTS.md

This is a Go terminal dashboard for AI service quotas. It is packaged as a Nix
flake and uses Bubble Tea + Lip Gloss for the TUI.

## Commands

- Enter dev shell: `nix develop`
- Run TUI: `go run ./cmd/quota`
- Run one-shot fetch: `go run ./cmd/quota once`
- Test: `nix develop --command go test ./...`
- Build package: `nix build`
- Format Go: `nix develop --command gofmt -w cmd internal`

## Project Shape

- `cmd/quota`: CLI entrypoint.
- `internal/provider`: provider auth, requests, parsing, and shared snapshot
  models.
- `internal/tui`: Bubble Tea model and Lip Gloss styles.
- `flake.nix`: Go toolchain, package, app, and dev shell.

## Development Notes

- Prefer existing local auth before env vars. Codex reads `~/.codex/auth.json`;
  Claude reads `~/.claude/.credentials.json`.
- Keep provider API details inside `internal/provider`; the TUI should render
  generic `Snapshot` and `Lane` values.
- Keep colors and layout in `internal/tui/theme.go` and rendering mechanics in
  `internal/tui/app.go`.
- Preserve the used/remaining display toggle and make new lane details work in
  both modes.
- Run `go test ./...` and `nix build` before handing off changes.
