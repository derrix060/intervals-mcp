# Intervals.icu MCP Server

## Overview
Go-based MCP (Model Context Protocol) server that auto-generates tools from the Intervals.icu OpenAPI spec. Zero-maintenance — tools update automatically when the API changes.

## Tech Stack
- Go 1.25+, `mark3labs/mcp-go` SDK, `kin-openapi` for spec parsing
- GoReleaser for builds (linux/darwin, amd64/arm64)
- Homebrew formula in `Formula/`

## Architecture
- `main.go` — Entry point, server setup
- `handler.go` — MCP tool handler, OpenAPI-to-MCP translation
- `openapi.go` — OpenAPI spec fetching and parsing
- `schema.go` — JSON Schema conversion
- `handler_test.go` — Tests

## Rules
- Run `go vet ./...` and `go test ./...` before any changes
- This is a small, focused codebase — keep it simple
- Tools are generated dynamically from OpenAPI spec — do not hardcode tool definitions
- Environment variables: `INTERVALS_API_KEY`, `INTERVALS_ATHLETE_ID`

## Release
- Tags trigger GoReleaser via `release.yml`
- Builds binaries + updates Homebrew formula automatically
