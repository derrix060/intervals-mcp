# intervals-mcp

A dynamic MCP (Model Context Protocol) server for [Intervals.icu](https://intervals.icu) — the training analytics platform for cyclists, runners, and triathletes.

The server reads the Intervals.icu OpenAPI specification at startup, automatically generates **144 tools** (as of Feb 2026) covering the entire API, and proxies tool calls to the real API. When the Intervals.icu API changes, simply restart the server to pick up the new endpoints — zero code changes required.

## Why

Intervals.icu has a rich API with 146 endpoints covering activities, wellness, workouts, power curves, gear, calendar events, and more. Using it from an LLM requires either hand-writing tool definitions for each endpoint or building a generic proxy.

This server takes the proxy approach: it fetches the official OpenAPI spec, converts every operation into an MCP tool with proper parameter schemas, and handles authentication and athlete ID injection automatically. The result is that any MCP-compatible client (Claude Code, Claude Desktop, etc.) gets full access to your Intervals.icu data without any manual tool definitions.

## Benefits

- **Full API coverage** — All non-file-upload endpoints are available as tools, generated directly from the official spec
- **Zero maintenance** — API changes are picked up on next restart with no code modifications
- **Automatic athlete ID injection** — The `athleteId` parameter is auto-filled on every call, so the LLM never needs to ask for it
- **Format flexibility** — Endpoints that support CSV, FIT, or other formats via `{ext}` are exposed with an optional `ext` parameter (defaults to JSON)
- **Tag filtering** — Expose only the endpoints you need by including or excluding OpenAPI tags
- **Single binary** — Built in Go, compiles to a single static binary with no runtime dependencies

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap derrix060/intervals-mcp https://github.com/derrix060/intervals-mcp
brew install intervals-mcp
```

### With `go install`

Requires [Go](https://go.dev/dl/) 1.21 or later.

```bash
go install github.com/derrix060/intervals-mcp@latest
```

This places the binary in your `$GOPATH/bin` (or `$HOME/go/bin` by default).

### From source

```bash
git clone https://github.com/derrix060/intervals-mcp.git
cd intervals-mcp
go build -o intervals-mcp .
```

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

## Comparison with other Intervals.icu MCP servers

There are several other MCP servers for Intervals.icu. All of them define tools manually, which means they cover only a fraction of the API and require code changes whenever endpoints are added or modified.

| Project | Language | Tools | Dynamic? | Last updated | Status |
|---|---|---|---|---|---|
| **This project** | **Go** | **144** | **Yes** | **2026-02** | **Active** |
| [mvilanova/intervals-mcp-server](https://github.com/mvilanova/intervals-mcp-server) | Python | 6 | No | 2025-12 | Stale |
| [like-a-freedom/rusty-intervals-mcp](https://github.com/like-a-freedom/rusty-intervals-mcp) | Rust | 57 | No | 2026-02 | Active |
| [eddmann/intervals-icu-mcp](https://github.com/eddmann/intervals-icu-mcp) | Python | 48 | No | 2025-11 | Stale |
| [gesteves/domestique](https://github.com/gesteves/domestique) | TypeScript | 43\* | No | 2026-02 | Active |
| [patrikmichi/intervals-icu-mcp](https://github.com/patrikmichi/intervals-icu-mcp) | TypeScript | ~30 | No | 2026-02 | New |
| [mrgeorgegray/intervals-icu-mcp](https://github.com/mrgeorgegray/intervals-icu-mcp) | TypeScript | 12 | No | 2025-07 | Stale |
| [notvincent/Intervals-ICU-MCP](https://github.com/notvincent/Intervals-ICU-MCP) | TypeScript | 4 | No | 2025-10 | Stale |

\* *domestique spans multiple platforms (Intervals.icu + Whoop + TrainerRoad + CORE), so not all 43 tools target Intervals.icu.*

### Key differences

- **Full API coverage**: The most comprehensive hardcoded server has 57 tools. This project exposes 144 — the entire API minus 2 file-upload endpoints that don't make sense for LLM tool use.
- **Zero maintenance**: Every hardcoded server falls behind when Intervals.icu adds or changes endpoints. This project reads the official OpenAPI spec at startup, so a restart is all it takes.
- **Tag filtering**: No other server lets you include or exclude groups of endpoints via tags.
- **Single binary, no runtime**: Like the Rust server, this compiles to a single binary. Unlike the Python and TypeScript servers, there is no interpreter, virtual environment, or `node_modules` to manage.

## How it works

1. On startup, the server fetches the OpenAPI 3.0 spec from `https://intervals.icu/api/v1/docs`
2. Each API operation is converted into an MCP tool using the `operationId` as the tool name
3. Path and query parameters become tool input properties with proper JSON Schema types
4. Request bodies are exposed as a single `body` parameter
5. Athlete ID parameters are detected and auto-injected from the environment variable
6. When the LLM calls a tool, the handler builds an HTTP request, authenticates with Basic Auth, and returns the response

## License

MIT
