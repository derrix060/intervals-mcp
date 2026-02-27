package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LoadSpec fetches and parses an OpenAPI 3.0 spec from the given URL.
func LoadSpec(specURL string) (*openapi3.T, error) {
	u, err := url.Parse(specURL)
	if err != nil {
		return nil, fmt.Errorf("invalid spec URL %q: %w", specURL, err)
	}
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromURI(u)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}
	return doc, nil
}

// GenerateTools converts all operations in the OpenAPI spec into MCP server tools.
func GenerateTools(doc *openapi3.T, cfg Config, client *http.Client) ([]server.ServerTool, error) {
	if doc.Paths == nil {
		return nil, fmt.Errorf("OpenAPI spec has no paths")
	}

	includeTags := parseTagSet(cfg.IncludeTags)
	excludeTags := parseTagSet(cfg.ExcludeTags)

	var tools []server.ServerTool
	skippedMultipart := 0
	skippedNoID := 0
	skippedFiltered := 0

	for pathStr, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}

		// Collect path-level parameters.
		pathLevelParams := pathItem.Parameters

		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			// Skip operations without an operationId.
			if op.OperationID == "" {
				skippedNoID++
				continue
			}

			// Skip if tag-filtered.
			if shouldFilterOp(op.Tags, includeTags, excludeTags) {
				skippedFiltered++
				continue
			}

			// Skip multipart/form-data endpoints (file uploads).
			if hasMultipartBody(op) {
				skippedMultipart++
				log.Printf("Skipping multipart endpoint: %s %s (%s)", method, pathStr, op.OperationID)
				continue
			}

			// Merge path-level + operation-level params, dedup by in:name.
			mergedParams := mergeParams(pathLevelParams, op.Parameters)

			// Classify parameters.
			athleteIDParams := map[string]bool{}
			hasExt := false
			var pathParams, queryParams []string

			for _, pRef := range mergedParams {
				if pRef == nil || pRef.Value == nil {
					continue
				}
				p := pRef.Value

				// Detect athlete ID params for auto-injection.
				if isAthleteIDParam(p, pathStr) {
					athleteIDParams[p.Name] = true
					continue
				}

				// Detect {ext} param.
				if p.Name == "ext" {
					hasExt = true
					continue
				}

				switch p.In {
				case "path":
					pathParams = append(pathParams, p.Name)
				case "query":
					queryParams = append(queryParams, p.Name)
				}
			}

			// Determine if the operation has a JSON request body.
			hasBody := false
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				jsonMedia := op.RequestBody.Value.Content.Get("application/json")
				if jsonMedia != nil {
					hasBody = true
				}
			}

			// Build the tool description.
			desc := op.Summary
			if desc == "" {
				desc = op.Description
			}
			if desc == "" {
				desc = fmt.Sprintf("%s %s", method, pathStr)
			}

			// Build MCP tool.
			inputSchema := buildInputSchema(mergedParams, op.RequestBody, athleteIDParams, hasExt)

			tool := mcp.Tool{
				Name:        op.OperationID,
				Description: desc,
				InputSchema: inputSchema,
			}

			info := operationInfo{
				Method:          method,
				PathPattern:     pathStr,
				PathParams:      pathParams,
				QueryParams:     queryParams,
				AthleteIDParams: athleteIDParams,
				HasBody:         hasBody,
				HasExt:          hasExt,
			}

			handler := makeHandler(info, cfg, client)

			tools = append(tools, server.ServerTool{
				Tool:    tool,
				Handler: handler,
			})
		}
	}

	log.Printf("Generated %d tools (skipped: %d no operationId, %d multipart, %d filtered)",
		len(tools), skippedNoID, skippedMultipart, skippedFiltered)

	return tools, nil
}

// isAthleteIDParam returns true if the parameter should be auto-injected with the athlete ID.
func isAthleteIDParam(p *openapi3.Parameter, pathStr string) bool {
	if p.In != "path" {
		return false
	}
	// Param named "athleteId" is always auto-injected.
	if p.Name == "athleteId" {
		return true
	}
	// Param named "id" in a path containing "/athlete/{id}" is auto-injected.
	if p.Name == "id" && strings.Contains(pathStr, "/athlete/{id}") {
		return true
	}
	return false
}

// hasMultipartBody returns true if the operation accepts multipart/form-data.
func hasMultipartBody(op *openapi3.Operation) bool {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return false
	}
	for mediaType := range op.RequestBody.Value.Content {
		if strings.Contains(mediaType, "multipart") {
			return true
		}
	}
	return false
}

// mergeParams merges path-level and operation-level parameters, deduplicating by in:name.
// Operation-level params override path-level params.
func mergeParams(pathParams, opParams openapi3.Parameters) openapi3.Parameters {
	seen := map[string]int{} // key: "in:name" → index in result
	var result openapi3.Parameters

	// Add path-level params first.
	for _, p := range pathParams {
		if p == nil || p.Value == nil {
			continue
		}
		key := p.Value.In + ":" + p.Value.Name
		seen[key] = len(result)
		result = append(result, p)
	}

	// Override with operation-level params.
	for _, p := range opParams {
		if p == nil || p.Value == nil {
			continue
		}
		key := p.Value.In + ":" + p.Value.Name
		if idx, exists := seen[key]; exists {
			result[idx] = p // Override
		} else {
			seen[key] = len(result)
			result = append(result, p)
		}
	}

	return result
}

// shouldFilterOp returns true if the operation should be excluded based on tag filters.
func shouldFilterOp(opTags []string, includeTags, excludeTags map[string]bool) bool {
	if len(includeTags) > 0 {
		// Only include if at least one tag matches.
		for _, t := range opTags {
			if includeTags[t] {
				return false
			}
		}
		return true
	}
	if len(excludeTags) > 0 {
		// Exclude if any tag matches.
		for _, t := range opTags {
			if excludeTags[t] {
				return true
			}
		}
	}
	return false
}

// parseTagSet splits a comma-separated tag string into a set.
func parseTagSet(tags string) map[string]bool {
	if tags == "" {
		return nil
	}
	set := map[string]bool{}
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			set[t] = true
		}
	}
	return set
}
