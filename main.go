package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// Config holds the server configuration read from environment variables.
type Config struct {
	APIKey      string
	AthleteID   string
	BaseURL     string
	IncludeTags string
	ExcludeTags string
}

const defaultBaseURL = "https://intervals.icu"
const specPath = "/api/v1/docs"

func main() {
	// All logging goes to stderr (stdout is the MCP transport).
	log.SetOutput(os.Stderr)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if cfg.IncludeTags != "" && cfg.ExcludeTags != "" {
		log.Fatal("Cannot set both INTERVALS_INCLUDE_TAGS and INTERVALS_EXCLUDE_TAGS")
	}

	specURL := cfg.BaseURL + specPath
	log.Printf("Loading OpenAPI spec from %s", specURL)

	doc, err := LoadSpec(specURL)
	if err != nil {
		log.Fatalf("Failed to load spec: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	tools, err := GenerateTools(doc, cfg, client)
	if err != nil {
		log.Fatalf("Failed to generate tools: %v", err)
	}

	mcpServer := server.NewMCPServer(
		"intervals-icu",
		"1.0.0",
	)

	mcpServer.AddTools(tools...)

	log.Printf("Starting intervals-icu MCP server with %d tools", len(tools))

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadConfig() (Config, error) {
	apiKey := os.Getenv("INTERVALS_API_KEY")
	if apiKey == "" {
		return Config{}, fmt.Errorf("INTERVALS_API_KEY environment variable is required")
	}

	athleteID := os.Getenv("INTERVALS_ATHLETE_ID")
	if athleteID == "" {
		return Config{}, fmt.Errorf("INTERVALS_ATHLETE_ID environment variable is required")
	}

	baseURL := os.Getenv("INTERVALS_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return Config{
		APIKey:      apiKey,
		AthleteID:   athleteID,
		BaseURL:     baseURL,
		IncludeTags: os.Getenv("INTERVALS_INCLUDE_TAGS"),
		ExcludeTags: os.Getenv("INTERVALS_EXCLUDE_TAGS"),
	}, nil
}
