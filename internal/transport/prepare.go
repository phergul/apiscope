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

func NewExecutor(client *http.Client) *Executor {
	if client == nil {
		client = http.DefaultClient
	}

	return &Executor{client: client}
}

func (e *Executor) PrepareRequest(operation *model.Operation, draft *model.RequestDraft, serverURL string) (*http.Request, error) {
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
	if draft != nil && draft.BodyRaw != "" {
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
		if draft.BodyRaw != "" && strings.TrimSpace(draft.BodyMediaType) != "" {
			request.Header.Set("Content-Type", draft.BodyMediaType)
		}
	}

	return request, nil
}

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
