package converter

import (
	"errors"
	"net/url"
	"strings"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerInfo(parsed *pipeline.ParsedDocument) *openapi3.Info {
	infoMap, _ := getMap(parsed.SwaggerDoc, "info")

	return &openapi3.Info{
		Title:          getString(infoMap, "title"),
		Description:    getString(infoMap, "description"),
		Version:        getString(infoMap, "version"),
		TermsOfService: getString(infoMap, "termsOfService"),
	}
}

func convertSwaggerServers(swaggerDoc map[string]any) (openapi3.Servers, error) {
	host := strings.TrimSpace(getString(swaggerDoc, "host"))
	basePath := strings.TrimSpace(getString(swaggerDoc, "basePath"))
	if basePath == "" {
		basePath = "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	schemes := getStringSlice(swaggerDoc, "schemes")
	if len(schemes) == 0 {
		schemes = []string{"https"}
	}

	if host == "" {
		return openapi3.Servers{
			&openapi3.Server{URL: basePath},
		}, nil
	}

	servers := make(openapi3.Servers, 0, len(schemes))
	for _, scheme := range schemes {
		if scheme == "" {
			continue
		}
		serverURL := (&url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   basePath,
		}).String()
		servers = append(servers, &openapi3.Server{URL: serverURL})
	}

	if len(servers) == 0 {
		return nil, &pipeline.Error{
			Kind:   pipeline.ErrorKindSwaggerConversionFailure,
			Op:     "convert servers",
			Source: getString(swaggerDoc, "host"),
			Err:    errors.New("swagger schemes did not produce any servers"),
		}
	}

	return servers, nil
}
