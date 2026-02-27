package main

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
)

const maxSchemaDepth = 3

// buildInputSchema constructs an MCP tool input schema from OpenAPI parameters and request body.
// It excludes athlete ID parameters (auto-injected) and handles {ext} as an optional param.
func buildInputSchema(params []*openapi3.ParameterRef, reqBody *openapi3.RequestBodyRef, athleteIDParams map[string]bool, hasExt bool) mcp.ToolInputSchema {
	properties := map[string]any{}
	var required []string

	for _, pRef := range params {
		if pRef == nil || pRef.Value == nil {
			continue
		}
		p := pRef.Value

		// Skip athlete ID params — they are auto-injected.
		if athleteIDParams[p.Name] {
			continue
		}

		// Handle {ext} param separately.
		if p.Name == "ext" {
			properties["ext"] = map[string]any{
				"type":        "string",
				"description": "Response format extension (e.g. \"csv\", \"fit\"). Leave empty for JSON.",
			}
			continue
		}

		prop := map[string]any{}
		if p.Schema != nil && p.Schema.Value != nil {
			prop = convertSchemaToMap(p.Schema.Value, 0)
		}
		if p.Description != "" {
			prop["description"] = p.Description
		}

		properties[p.Name] = prop

		if p.Required {
			required = append(required, p.Name)
		}
	}

	// Request body → single "body" property.
	if reqBody != nil && reqBody.Value != nil {
		rb := reqBody.Value
		jsonMedia := rb.Content.Get("application/json")
		if jsonMedia != nil && jsonMedia.Schema != nil && jsonMedia.Schema.Value != nil {
			bodyProp := convertSchemaToMap(jsonMedia.Schema.Value, 0)
			if rb.Description != "" {
				bodyProp["description"] = rb.Description
			}
			properties["body"] = bodyProp
			if rb.Required {
				required = append(required, "body")
			}
		}
	}

	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

// convertSchemaToMap recursively converts an OpenAPI Schema to a JSON Schema map.
// Depth is limited to maxSchemaDepth to prevent explosion on deeply nested schemas.
func convertSchemaToMap(s *openapi3.Schema, depth int) map[string]any {
	if s == nil {
		return map[string]any{}
	}

	m := map[string]any{}

	// Type
	if s.Type != nil {
		types := []string(*s.Type)
		if len(types) == 1 {
			m["type"] = types[0]
		} else if len(types) > 1 {
			m["type"] = types
		}
	}

	// Basic metadata
	if s.Format != "" {
		m["format"] = s.Format
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Default != nil {
		m["default"] = s.Default
	}

	// Numeric constraints
	if s.Min != nil {
		m["minimum"] = *s.Min
	}
	if s.Max != nil {
		m["maximum"] = *s.Max
	}

	// String constraints
	if s.MinLength > 0 {
		m["minLength"] = s.MinLength
	}
	if s.MaxLength != nil {
		m["maxLength"] = *s.MaxLength
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}

	// Stop expanding nested structures at max depth.
	if depth >= maxSchemaDepth {
		return m
	}

	// Object properties
	if len(s.Properties) > 0 {
		props := map[string]any{}
		for name, propRef := range s.Properties {
			if propRef != nil && propRef.Value != nil {
				props[name] = convertSchemaToMap(propRef.Value, depth+1)
			}
		}
		m["properties"] = props
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}

	// Array items
	if s.Items != nil && s.Items.Value != nil {
		m["items"] = convertSchemaToMap(s.Items.Value, depth+1)
	}

	// Composition keywords
	if len(s.AllOf) > 0 {
		if len(s.AllOf) == 1 && s.AllOf[0].Value != nil {
			// Flatten single allOf.
			inner := convertSchemaToMap(s.AllOf[0].Value, depth+1)
			for k, v := range inner {
				if _, exists := m[k]; !exists {
					m[k] = v
				}
			}
		} else {
			allOf := make([]any, 0, len(s.AllOf))
			for _, ref := range s.AllOf {
				if ref.Value != nil {
					allOf = append(allOf, convertSchemaToMap(ref.Value, depth+1))
				}
			}
			if len(allOf) > 0 {
				m["allOf"] = allOf
			}
		}
	}
	if len(s.AnyOf) > 0 {
		anyOf := make([]any, 0, len(s.AnyOf))
		for _, ref := range s.AnyOf {
			if ref.Value != nil {
				anyOf = append(anyOf, convertSchemaToMap(ref.Value, depth+1))
			}
		}
		if len(anyOf) > 0 {
			m["anyOf"] = anyOf
		}
	}
	if len(s.OneOf) > 0 {
		oneOf := make([]any, 0, len(s.OneOf))
		for _, ref := range s.OneOf {
			if ref.Value != nil {
				oneOf = append(oneOf, convertSchemaToMap(ref.Value, depth+1))
			}
		}
		if len(oneOf) > 0 {
			m["oneOf"] = oneOf
		}
	}

	return m
}
