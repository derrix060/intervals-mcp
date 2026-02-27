# intervals-mcp

A dynamic MCP (Model Context Protocol) server for [Intervals.icu](https://intervals.icu) — the training analytics platform for cyclists, runners, and triathletes.

The server reads the Intervals.icu OpenAPI specification at startup, automatically generates **144 tools** covering the entire API, and proxies tool calls to the real API. When the Intervals.icu API changes, simply restart the server to pick up the new endpoints — zero code changes required.

## Why

Intervals.icu has a rich API with 146 endpoints covering activities, wellness, workouts, power curves, gear, calendar events, and more. Using it from an LLM requires either hand-writing tool definitions for each endpoint or building a generic proxy.

This server takes the proxy approach: it fetches the official OpenAPI spec, converts every operation into an MCP tool with proper parameter schemas, and handles authentication and athlete ID injection automatically. The result is that any MCP-compatible client (Claude Code, Claude Desktop, etc.) gets full access to your Intervals.icu data without any manual tool definitions.

## Benefits

- **Full API coverage** — All 144 non-file-upload endpoints are available as tools, generated directly from the official spec
- **Zero maintenance** — API changes are picked up on next restart with no code modifications
- **Automatic athlete ID injection** — The `athleteId` parameter is auto-filled on every call, so the LLM never needs to ask for it
- **Format flexibility** — Endpoints that support CSV, FIT, or other formats via `{ext}` are exposed with an optional `ext` parameter (defaults to JSON)
- **Tag filtering** — Expose only the endpoints you need by including or excluding OpenAPI tags
- **Single binary** — Built in Go, compiles to a single static binary with no runtime dependencies

## Prerequisites

- [Go](https://go.dev/dl/) 1.21 or later
- An Intervals.icu account with an API key (Settings > Developer Settings)

## Installation

### From source

```bash
git clone https://github.com/derrix060/intervals-mcp.git
cd intervals-mcp
go build -o intervals-mcp .
```

The binary is now at `./intervals-mcp`.

### With `go install`

```bash
go install github.com/derrix060/intervals-mcp@latest
```

This places the binary in your `$GOPATH/bin` (or `$HOME/go/bin` by default).

## Configuration

The server is configured entirely through environment variables.

| Variable | Required | Description |
|---|---|---|
| `INTERVALS_API_KEY` | Yes | Your Intervals.icu API key |
| `INTERVALS_ATHLETE_ID` | Yes | Your athlete ID (e.g. `i12345`) |
| `INTERVALS_BASE_URL` | No | API base URL (default: `https://intervals.icu`) |
| `INTERVALS_INCLUDE_TAGS` | No | Comma-separated tags to include (e.g. `Wellness,Activities`) |
| `INTERVALS_EXCLUDE_TAGS` | No | Comma-separated tags to exclude |

`INTERVALS_INCLUDE_TAGS` and `INTERVALS_EXCLUDE_TAGS` are mutually exclusive — set one or neither, not both.

### Finding your credentials

1. Log in to [Intervals.icu](https://intervals.icu)
2. Go to **Settings > Developer Settings**
3. Generate an API key — this is your `INTERVALS_API_KEY`
4. Your athlete ID is shown on the same page (format: `i12345`) — this is your `INTERVALS_ATHLETE_ID`

## Usage with Claude Code

Add to your Claude Code MCP configuration (`~/.claude/claude_desktop_config.json` or via `claude mcp add`):

```json
{
  "mcpServers": {
    "intervals-icu": {
      "command": "/path/to/intervals-mcp",
      "env": {
        "INTERVALS_API_KEY": "your-api-key",
        "INTERVALS_ATHLETE_ID": "i12345"
      }
    }
  }
}
```

Or using the CLI:

```bash
claude mcp add intervals-icu /path/to/intervals-mcp \
  -e INTERVALS_API_KEY=your-api-key \
  -e INTERVALS_ATHLETE_ID=i12345
```

Once configured, you can ask Claude things like:

- "Show me my recent activities"
- "What's my current fitness summary?"
- "Get my wellness data for this week"
- "List my upcoming workouts"
- "Show my power curve for the last 90 days"

## Usage with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "intervals-icu": {
      "command": "/path/to/intervals-mcp",
      "env": {
        "INTERVALS_API_KEY": "your-api-key",
        "INTERVALS_ATHLETE_ID": "i12345"
      }
    }
  }
}
```

## Tag filtering

To expose only a subset of the API, use tag filtering. Tags correspond to the sections in the [Intervals.icu API docs](https://intervals.icu/api/v1/docs/swagger-ui/index.html).

Include only wellness and activity endpoints:

```bash
INTERVALS_INCLUDE_TAGS="Wellness,Activities"
```

Exclude gear-related endpoints:

```bash
INTERVALS_EXCLUDE_TAGS="Gear"
```

## How it works

1. On startup, the server fetches the OpenAPI 3.0 spec from `https://intervals.icu/api/v1/docs`
2. Each API operation is converted into an MCP tool using the `operationId` as the tool name
3. Path and query parameters become tool input properties with proper JSON Schema types
4. Request bodies are exposed as a single `body` parameter
5. Athlete ID parameters are detected and auto-injected from the environment variable
6. When the LLM calls a tool, the handler builds an HTTP request, authenticates with Basic Auth, and returns the response

## License

MIT
