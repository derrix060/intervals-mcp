package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// formatValue converts a value to a string suitable for URL path/query params.
// JSON numbers are unmarshaled as float64 in Go; large integers like 95893899
// would format as "9.5893899e+07" with %v, breaking API URLs.
func formatValue(v any) string {
	if f, ok := v.(float64); ok && f == math.Trunc(f) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", v)
}

// operationInfo captures everything needed to proxy a single API call.
type operationInfo struct {
	Method          string          // HTTP method (GET, POST, etc.)
	PathPattern     string          // e.g. "/api/v1/athlete/{id}/activities"
	PathParams      []string        // path param names (excluding athlete ID)
	QueryParams     []string        // query param names
	AthleteIDParams map[string]bool // param names that should be auto-injected
	HasBody         bool            // whether the operation accepts a JSON body
	HasExt          bool            // whether the path has an {ext} param
}

// makeHandler creates an MCP tool handler that proxies calls to the Intervals.icu API.
func makeHandler(info operationInfo, cfg Config, client *http.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Build the URL path by substituting path parameters.
		path := info.PathPattern
		for _, name := range info.PathParams {
			val := formatValue(args[name])
			path = strings.ReplaceAll(path, "{"+name+"}", val)
		}

		// Auto-inject athlete ID parameters.
		for paramName := range info.AthleteIDParams {
			path = strings.ReplaceAll(path, "{"+paramName+"}", cfg.AthleteID)
		}

		// Handle {ext} parameter.
		if info.HasExt {
			ext := ""
			if v, ok := args["ext"]; ok {
				ext = fmt.Sprintf("%v", v)
			}
			if ext == "" {
				// Remove the {ext} placeholder entirely (default to JSON).
				path = strings.ReplaceAll(path, "{ext}", "")
			} else {
				// Ensure dot prefix.
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				path = strings.ReplaceAll(path, "{ext}", ext)
			}
		}

		// Build query string.
		q := url.Values{}
		for _, name := range info.QueryParams {
			v, ok := args[name]
			if !ok {
				continue
			}
			switch val := v.(type) {
			case []any:
				for _, item := range val {
					q.Add(name, formatValue(item))
				}
			default:
				q.Set(name, formatValue(val))
			}
		}

		fullURL := cfg.BaseURL + path
		if encoded := q.Encode(); encoded != "" {
			fullURL += "?" + encoded
		}

		// Build request body if present.
		var bodyReader io.Reader
		if info.HasBody {
			if bodyArg, ok := args["body"]; ok {
				bodyBytes, err := json.Marshal(bodyArg)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to marshal request body: %v", err)), nil
				}
				bodyReader = bytes.NewReader(bodyBytes)
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, info.Method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Basic Auth: API_KEY as username, api_key as password.
		httpReq.SetBasicAuth("API_KEY", cfg.APIKey)

		if info.HasBody && bodyReader != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("API request failed: %v", err)), nil
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read response: %v", err)), nil
		}

		// 204 No Content.
		if resp.StatusCode == http.StatusNoContent {
			return mcp.NewToolResultText("Success (204 No Content)"), nil
		}

		// Error responses.
		if resp.StatusCode >= 400 {
			return mcp.NewToolResultError(fmt.Sprintf("API error %d: %s", resp.StatusCode, string(respBody))), nil
		}

		// Try to pretty-print JSON responses.
		var jsonData any
		if err := json.Unmarshal(respBody, &jsonData); err == nil {
			pretty, err := json.MarshalIndent(jsonData, "", "  ")
			if err == nil {
				return mcp.NewToolResultText(string(pretty)), nil
			}
		}

		// Return raw response if not JSON.
		return mcp.NewToolResultText(string(respBody)), nil
	}
}
