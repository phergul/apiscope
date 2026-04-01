package transport

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

type Executor struct {
	client *http.Client
}

// NewExecutor builds a transport executor with the provided HTTP client.
func NewExecutor(client *http.Client) *Executor {
	if client == nil {
		client = http.DefaultClient
	}

	return &Executor{client: client}
}

// PrepareRequest builds an HTTP request from the selected operation and draft inputs.
func (e *Executor) PrepareRequest(
	operation *model.Operation,
	draft *model.RequestDraft,
	serverURL string,
	requirement *model.SecurityRequirement,
	securitySchemes map[string]model.SecurityScheme,
	authState map[string]model.AuthValue,
) (*http.Request, error) {
	if operation == nil {
		return nil, errors.New("no operation selected")
	}
	if strings.TrimSpace(serverURL) == "" {
		return nil, errors.New("no server selected")
	}

	baseURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	resolvedPath, err := resolvePath(operation.Path, draft)
	if err != nil {
		return nil, err
	}

	targetURL := baseURL.ResolveReference(&url.URL{Path: resolvedPath})
	queryValues := targetURL.Query()
	if draft != nil {
		for key, value := range draft.QueryParams {
			if strings.TrimSpace(value) == "" {
				continue
			}
			queryValues.Set(key, value)
		}
	}
	targetURL.RawQuery = queryValues.Encode()

	var body io.Reader
	formBody := ""
	if operation.FormBodyMediaType == "application/x-www-form-urlencoded" {
		formBody = encodeFormParams(draft)
		body = strings.NewReader(formBody)
	} else if draft != nil && draft.BodyRaw != "" {
		body = strings.NewReader(draft.BodyRaw)
	}

	request, err := http.NewRequest(strings.ToUpper(operation.Method), targetURL.String(), body)
	if err != nil {
		return nil, err
	}

	if draft != nil {
		for key, value := range draft.HeaderParams {
			if strings.TrimSpace(value) == "" {
				continue
			}
			request.Header.Set(key, value)
		}
		for key, value := range draft.CookieParams {
			if strings.TrimSpace(value) == "" {
				continue
			}
			request.AddCookie(&http.Cookie{Name: key, Value: value})
		}
		if operation.FormBodyMediaType == "application/x-www-form-urlencoded" {
			request.Header.Set("Content-Type", operation.FormBodyMediaType)
		} else if draft.BodyRaw != "" && strings.TrimSpace(draft.BodyMediaType) != "" {
			request.Header.Set("Content-Type", draft.BodyMediaType)
		}
	}
	if err := applyAuth(request, requirement, securitySchemes, authState); err != nil {
		return nil, err
	}

	return request, nil
}

// encodeFormParams serializes non-empty form values for urlencoded request bodies.
func encodeFormParams(draft *model.RequestDraft) string {
	if draft == nil || len(draft.FormParams) == 0 {
		return ""
	}

	values := url.Values{}
	for key, value := range draft.FormParams {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values.Set(key, value)
	}

	return values.Encode()
}

// resolvePath substitutes draft path parameters into an operation path template.
func resolvePath(path string, draft *model.RequestDraft) (string, error) {
	resolved := path
	if draft != nil {
		for key, value := range draft.PathParams {
			resolved = strings.ReplaceAll(resolved, "{"+key+"}", url.PathEscape(value))
		}
	}
	if strings.Contains(resolved, "{") || strings.Contains(resolved, "}") {
		return "", errors.New("path template contains unresolved parameters")
	}

	return resolved, nil
}
