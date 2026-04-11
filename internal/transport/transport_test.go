package transport

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestPrepareRequestBuildsPathQueryHeadersCookiesAndBody(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/pets/{petId}",
	}
	draft := &model.RequestDraft{
		PathParams:    map[string]string{"petId": "abc"},
		QueryParams:   map[string]string{"limit": "10"},
		HeaderParams:  map[string]string{"X-Trace-ID": "trace-1"},
		CookieParams:  map[string]string{"session": "cookie-1"},
		BodyMediaType: "application/json",
		BodyRaw:       `{"name":"fido"}`,
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.URL.String(); got != "https://api.example.com/pets/abc?limit=10" {
		t.Fatalf("unexpected prepared URL %q", got)
	}
	if got := request.Header.Get("X-Trace-ID"); got != "trace-1" {
		t.Fatalf("expected header to be set, got %q", got)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type to be set, got %q", got)
	}
	cookie, err := request.Cookie("session")
	if err != nil {
		t.Fatalf("expected cookie to be set: %v", err)
	}
	if cookie.Value != "cookie-1" {
		t.Fatalf("expected cookie value cookie-1, got %q", cookie.Value)
	}
}

func TestPrepareRequestSerializesUrlencodedFormParams(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method:            "POST",
		Path:              "/pets",
		FormBodyMediaType: "application/x-www-form-urlencoded",
	}
	draft := &model.RequestDraft{
		QueryParams:  map[string]string{"limit": "10"},
		HeaderParams: map[string]string{"X-Trace-ID": "trace-1"},
		CookieParams: map[string]string{"session": "cookie-1"},
		FormParams:   map[string]string{"name": "fido", "skip": "  "},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form content type, got %q", got)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if got := string(body); got != "name=fido" {
		t.Fatalf("expected urlencoded form body, got %q", got)
	}
	if got := request.URL.String(); got != "https://api.example.com/pets?limit=10" {
		t.Fatalf("unexpected prepared URL %q", got)
	}
	if got := request.Header.Get("X-Trace-ID"); got != "trace-1" {
		t.Fatalf("expected header to be preserved, got %q", got)
	}
	cookie, err := request.Cookie("session")
	if err != nil {
		t.Fatalf("expected cookie to be set: %v", err)
	}
	if cookie.Value != "cookie-1" {
		t.Fatalf("expected cookie value cookie-1, got %q", cookie.Value)
	}
}

func TestPrepareRequestSerializesMultipartFormParams(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method:            "POST",
		Path:              "/pets",
		FormBodyMediaType: "multipart/form-data",
	}
	draft := &model.RequestDraft{
		FormParams: map[string]string{"name": "fido", "skip": "   "},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart form content type, got %q", mediaType)
	}
	reader := multipart.NewReader(request.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("NextPart returned error: %v", err)
	}
	if got := part.FormName(); got != "name" {
		t.Fatalf("expected form field part, got %q", got)
	}
	body, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if got := string(body); got != "fido" {
		t.Fatalf("expected multipart form value, got %q", got)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("expected one multipart field part, got %v", err)
	}
}

func TestPrepareRequestSerializesMultipartFileUpload(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "avatar.txt")
	if err := os.WriteFile(uploadPath, []byte("hello file"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method:            "POST",
		Path:              "/upload",
		FormBodyMediaType: "multipart/form-data",
	}
	draft := &model.RequestDraft{
		FormParams:     map[string]string{"description": "avatar"},
		FormFileParams: map[string]string{"file": uploadPath},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart form content type, got %q", mediaType)
	}

	reader := multipart.NewReader(request.Body, params["boundary"])
	parts := map[string]string{}
	filenames := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart returned error: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		parts[part.FormName()] = string(body)
		filenames[part.FormName()] = part.FileName()
	}
	if got := parts["description"]; got != "avatar" {
		t.Fatalf("expected multipart scalar field, got %q", got)
	}
	if got := parts["file"]; got != "hello file" {
		t.Fatalf("expected multipart file part, got %q", got)
	}
	if got := filenames["file"]; got != "avatar.txt" {
		t.Fatalf("expected multipart filename avatar.txt, got %q", got)
	}
}

func TestPrepareRequestSerializesMultipartRequestBodyFields(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "avatar.txt")
	if err := os.WriteFile(uploadPath, []byte("hello file"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/upload",
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{MediaType: "multipart/form-data"}},
		},
	}
	draft := &model.RequestDraft{
		BodyMediaType:  "multipart/form-data",
		FormParams:     map[string]string{"description": "avatar"},
		FormFileParams: map[string]string{"file": uploadPath},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart request body content type, got %q", mediaType)
	}

	reader := multipart.NewReader(request.Body, params["boundary"])
	parts := map[string]string{}
	filenames := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart returned error: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		parts[part.FormName()] = string(body)
		filenames[part.FormName()] = part.FileName()
	}
	if got := parts["description"]; got != "avatar" {
		t.Fatalf("expected multipart request-body scalar field, got %q", got)
	}
	if got := parts["file"]; got != "hello file" {
		t.Fatalf("expected multipart request-body file part, got %q", got)
	}
	if got := filenames["file"]; got != "avatar.txt" {
		t.Fatalf("expected multipart request-body filename avatar.txt, got %q", got)
	}
}

func TestPrepareRequestSerializesStructuredMultipartRequestBodyFieldAsJSONPart(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/upload",
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "multipart/form-data",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"metadata": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"region": {Type: "string"},
							},
						},
					},
				},
			}},
		},
	}
	draft := &model.RequestDraft{
		BodyMediaType: "multipart/form-data",
		FormParams:    map[string]string{"metadata": `{"region":"ie"}`},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart request body content type, got %q", mediaType)
	}

	reader := multipart.NewReader(request.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("NextPart returned error: %v", err)
	}
	body, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if got := part.FormName(); got != "metadata" {
		t.Fatalf("expected structured multipart field name, got %q", got)
	}
	if got := part.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected structured multipart field content type, got %q", got)
	}
	if got := string(body); got != `{"region":"ie"}` {
		t.Fatalf("expected structured multipart field body, got %q", got)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("expected one structured multipart field part, got %v", err)
	}
}

func TestPrepareRequestAppliesMultipartEncodingContentTypeAndHeaders(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/upload",
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "multipart/form-data",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"metadata": {Type: "string"},
					},
				},
				Encoding: map[string]model.MediaTypeEncoding{
					"metadata": {
						PropertyName: "metadata",
						ContentType:  "application/merge-patch+json",
						Headers: []model.Parameter{{
							Name:    "X-Part-Trace",
							Example: "trace-1",
						}},
					},
				},
			}},
		},
	}
	draft := &model.RequestDraft{
		BodyMediaType: "multipart/form-data",
		FormParams:    map[string]string{"metadata": `{"region":"ie"}`},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart request body content type, got %q", mediaType)
	}

	reader := multipart.NewReader(request.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("NextPart returned error: %v", err)
	}
	if got := part.FormName(); got != "metadata" {
		t.Fatalf("expected metadata part, got %q", got)
	}
	if got := part.Header.Get("Content-Type"); got != "application/merge-patch+json" {
		t.Fatalf("expected encoded part content type, got %q", got)
	}
	if got := part.Header.Get("X-Part-Trace"); got != "trace-1" {
		t.Fatalf("expected encoded part header, got %q", got)
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("expected one multipart part, got %v", err)
	}
}

func TestPrepareRequestAppliesDraftMultipartEncodingOverride(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/upload",
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "multipart/form-data",
				Schema:    &model.Schema{Type: "object", Properties: map[string]*model.Schema{"metadata": {Type: "string"}}},
				Encoding: map[string]model.MediaTypeEncoding{
					"metadata": {PropertyName: "metadata", ContentType: "application/merge-patch+json"},
				},
			}},
		},
	}
	draft := &model.RequestDraft{
		BodyMediaType:    "multipart/form-data",
		FormParams:       map[string]string{"metadata": `{"region":"ie"}`},
		BodyPartEncoding: map[string]string{"metadata": "application/json"},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType returned error: %v", err)
	}
	if mediaType != "multipart/form-data" {
		t.Fatalf("expected multipart request body content type, got %q", mediaType)
	}

	reader := multipart.NewReader(request.Body, params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("NextPart returned error: %v", err)
	}
	if got := part.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected draft override content type on multipart part, got %q", got)
	}
}

func TestPrepareRequestSerializesUrlencodedRequestBodyFields(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/submit",
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{MediaType: "application/x-www-form-urlencoded"}},
		},
	}
	draft := &model.RequestDraft{
		BodyMediaType: "application/x-www-form-urlencoded",
		FormParams:    map[string]string{"attachment": "inline-data"},
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Fatalf("expected urlencoded request-body content type, got %q", got)
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if got := string(body); got != "attachment=inline-data" {
		t.Fatalf("expected urlencoded request-body fields, got %q", got)
	}
}

func TestPrepareRequestReturnsClearErrorForUnreadableMultipartFilePath(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{
		Method:            "POST",
		Path:              "/upload",
		FormBodyMediaType: "multipart/form-data",
	}
	draft := &model.RequestDraft{
		FormFileParams: map[string]string{"file": filepath.Join(t.TempDir(), "missing.txt")},
	}

	_, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err == nil {
		t.Fatal("expected unreadable file path error")
	}
	if !strings.Contains(err.Error(), `file form parameter "file" path "`) {
		t.Fatalf("expected file-path error message, got %v", err)
	}
}

func TestPrepareRequestAppliesSupportedAuth(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	operation := &model.Operation{Method: "GET", Path: "/me"}
	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
		},
	}
	request, err := executor.PrepareRequest(
		operation,
		&model.RequestDraft{},
		"https://api.example.com",
		requirement,
		map[string]model.SecurityScheme{
			"bearer_auth": {
				Name:   "bearer_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
		map[string]model.AuthValue{
			"bearer_auth": {Type: model.AuthSchemeValueTypeBearer, BearerToken: "token-123"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("expected bearer auth header, got %q", got)
	}
}

func TestPrepareRequestAppliesBasicAuth(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	request, err := executor.PrepareRequest(
		&model.Operation{Method: "GET", Path: "/me"},
		&model.RequestDraft{},
		"https://api.example.com",
		&model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "basic_auth"}}},
			},
		},
		map[string]model.SecurityScheme{
			"basic_auth": {
				Name:   "basic_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBasic,
			},
		},
		map[string]model.AuthValue{
			"basic_auth": {Type: model.AuthSchemeValueTypeBasic, Username: "alice", Password: "secret"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", got)
	}
}

func TestPrepareRequestAppliesQueryAPIKey(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	request, err := executor.PrepareRequest(
		&model.Operation{Method: "GET", Path: "/me"},
		&model.RequestDraft{},
		"https://api.example.com",
		&model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "query_key"}}},
			},
		},
		map[string]model.SecurityScheme{
			"query_key": {
				Name:          "query_key",
				Type:          model.SecuritySchemeTypeAPIKey,
				In:            model.ParameterLocationQuery,
				ParameterName: "api_key",
			},
		},
		map[string]model.AuthValue{
			"query_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.URL.Query().Get("api_key"); got != "secret" {
		t.Fatalf("expected query api key, got %q", got)
	}
}

func TestExportCurlRendersJSONRequest(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	command, err := executor.ExportCurl(
		&model.Operation{Method: "POST", Path: "/pets"},
		&model.RequestDraft{
			BodyMediaType: "application/json",
			BodyRaw:       `{"name":"fido"}`,
			HeaderParams:  map[string]string{"X-Trace-ID": "trace-1"},
		},
		"https://api.example.com",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ExportCurl returned error: %v", err)
	}
	for _, snippet := range []string{
		"curl \\",
		"-X 'POST'",
		"-H 'Content-Type: application/json'",
		"-H 'X-Trace-Id: trace-1'",
		"--data-raw '{\"name\":\"fido\"}'",
		"'https://api.example.com/pets'",
	} {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected curl snippet %q, got %q", snippet, command)
		}
	}
}

func TestExportCurlRendersMultipartFormParts(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "avatar.txt")
	if err := os.WriteFile(uploadPath, []byte("hello file"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	executor := NewExecutor(nil, nil)
	command, err := executor.ExportCurl(
		&model.Operation{
			Method: "POST",
			Path:   "/upload",
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{
					MediaType: "multipart/form-data",
					Schema: &model.Schema{
						Type: "object",
						Properties: map[string]*model.Schema{
							"description": {Type: "string"},
							"metadata": {
								Type:       "object",
								Properties: map[string]*model.Schema{"region": {Type: "string"}},
							},
							"file": {Type: "string", Format: "binary"},
						},
					},
				}},
			},
		},
		&model.RequestDraft{
			BodyMediaType:  "multipart/form-data",
			FormParams:     map[string]string{"description": "avatar", "metadata": `{"region":"ie"}`},
			FormFileParams: map[string]string{"file": uploadPath},
		},
		"https://api.example.com",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ExportCurl returned error: %v", err)
	}
	for _, snippet := range []string{
		"-F 'description=avatar'",
		"-F 'metadata={\"region\":\"ie\"};type=application/json'",
		"-F 'file=@" + uploadPath + "'",
		"'https://api.example.com/upload'",
	} {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected multipart curl snippet %q, got %q", snippet, command)
		}
	}
	if strings.Contains(command, "boundary=") {
		t.Fatalf("expected curl export to avoid prepared multipart boundary header, got %q", command)
	}
}

func TestExportCurlRendersMultipartEncodingContentTypeOverrides(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil, nil)
	command, err := executor.ExportCurl(
		&model.Operation{
			Method: "POST",
			Path:   "/upload",
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{
					MediaType: "multipart/form-data",
					Schema:    &model.Schema{Type: "object", Properties: map[string]*model.Schema{"metadata": {Type: "string"}}},
					Encoding: map[string]model.MediaTypeEncoding{
						"metadata": {PropertyName: "metadata", ContentType: "application/merge-patch+json"},
					},
				}},
			},
		},
		&model.RequestDraft{
			BodyMediaType:    "multipart/form-data",
			FormParams:       map[string]string{"metadata": `{"region":"ie"}`},
			BodyPartEncoding: map[string]string{"metadata": "application/json"},
		},
		"https://api.example.com",
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ExportCurl returned error: %v", err)
	}
	if !strings.Contains(command, `-F 'metadata={"region":"ie"};type=application/json'`) {
		t.Fatalf("expected curl multipart part to render draft encoding override, got %q", command)
	}
}

func TestExecuteCapturesHTTPResponseAndPrettyPrintsJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	executor := NewExecutor(server.Client(), nil)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/ping"), request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", response.StatusCode)
	}
	if response.ContentType != "application/json" {
		t.Fatalf("expected normalised content type, got %q", response.ContentType)
	}
	if !containsAll(response.PrettyBody, []string{"{", "\"ok\": true"}) {
		t.Fatalf("expected pretty body to contain formatted JSON, got %q", response.PrettyBody)
	}
}

func TestExecuteReturnsTransportErrorOnNetworkFailure(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(&http.Client{}, nil)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/fail"), request)
	if response.TransportError == "" {
		t.Fatal("expected transport error to be captured")
	}
}

func containsAll(value string, snippets []string) bool {
	for _, snippet := range snippets {
		if !strings.Contains(value, snippet) {
			return false
		}
	}

	return true
}
