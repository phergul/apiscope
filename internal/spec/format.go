package spec

import (
	"encoding/json"
	"mime"
	"path"
	"strings"

	"api-tui/internal/spec/internal/pipeline"
)

type DocumentFormat = pipeline.DocumentFormat

const (
	DocumentFormatJSON = pipeline.DocumentFormatJSON
	DocumentFormatYAML = pipeline.DocumentFormatYAML
)

func formatFromLocation(location string) (DocumentFormat, bool) {
	ext := strings.ToLower(path.Ext(location))
	switch ext {
	case ".json":
		return DocumentFormatJSON, true
	case ".yaml", ".yml":
		return DocumentFormatYAML, true
	default:
		return "", false
	}
}

func formatFromContentType(contentType string) (DocumentFormat, bool) {
	if contentType == "" {
		return "", false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", false
	}

	switch strings.ToLower(mediaType) {
	case "application/json", "application/problem+json":
		return DocumentFormatJSON, true
	case "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml":
		return DocumentFormatYAML, true
	default:
		return "", false
	}
}

func formatFromContent(raw []byte) (DocumentFormat, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", false
	}

	if json.Valid([]byte(trimmed)) {
		return DocumentFormatJSON, true
	}

	if looksLikeYAML(trimmed) {
		return DocumentFormatYAML, true
	}

	return "", false
}

func looksLikeYAML(content string) bool {
	if strings.HasPrefix(content, "---") || strings.HasPrefix(content, "%YAML") {
		return true
	}

	for _, prefix := range []string{"openapi:", "swagger:", "info:", "paths:", "components:", "servers:"} {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, ": ") || strings.HasSuffix(trimmed, ":") {
			return true
		}
	}

	return false
}
