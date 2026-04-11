package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// ExportCurl renders a curl command from the same prepared request path used for execution.
func (e *Executor) ExportCurl(
	operation *model.Operation,
	draft *model.RequestDraft,
	serverURL string,
	requirement *model.SecurityRequirement,
	securitySchemes map[string]model.SecurityScheme,
	authState map[string]model.AuthValue,
) (string, error) {
	request, err := e.PrepareRequest(operation, draft, serverURL, requirement, securitySchemes, authState)
	if err != nil {
		return "", err
	}

	return renderCurlCommand(operation, draft, request)
}

func renderCurlCommand(operation *model.Operation, draft *model.RequestDraft, request *http.Request) (string, error) {
	if request == nil {
		return "", fmt.Errorf("no request prepared")
	}

	lines := []string{"curl"}
	lines = append(lines, renderCurlFlag("-X", request.Method))

	for _, header := range sortedHeaderNames(request.Header) {
		if skipCurlHeader(operation, draft, header) {
			continue
		}
		for _, value := range request.Header.Values(header) {
			lines = append(lines, renderCurlFlag("-H", header+": "+value))
		}
	}

	if effectiveBodyMediaType(operation, draft) == "multipart/form-data" {
		lines = append(lines, renderMultipartCurlArgs(operation, draft)...)
	} else if body, err := readRequestBody(request); err != nil {
		return "", err
	} else if len(body) > 0 {
		lines = append(lines, renderCurlFlag("--data-raw", string(body)))
	}

	lines = append(lines, "  "+shellQuote(request.URL.String()))
	return strings.Join(lines, " \\\n"), nil
}

func renderMultipartCurlArgs(operation *model.Operation, draft *model.RequestDraft) []string {
	lines := make([]string, 0, len(draftFormValues(draft))+len(draftFileValues(draft)))
	jsonFields := multipartJSONFields(operation, draft)
	encodingByField := multipartEncodingByField(operation, draft)
	for _, key := range sortedKeys(draftFormValues(draft)) {
		value := strings.TrimSpace(draft.FormParams[key])
		if value == "" {
			continue
		}
		field := key + "=" + value
		contentType := multipartPartContentTypeForCurl(key, encodingByField, jsonFields)
		if contentType != "" {
			field += ";type=" + contentType
		}
		lines = append(lines, renderCurlFlag("-F", field))
	}
	for _, key := range sortedKeys(draftFileValues(draft)) {
		path := strings.TrimSpace(draft.FormFileParams[key])
		if path == "" {
			continue
		}
		field := key + "=@" + path
		if contentType := multipartPartContentTypeForCurl(key, encodingByField, jsonFields); contentType != "" {
			field += ";type=" + contentType
		}
		lines = append(lines, renderCurlFlag("-F", field))
	}
	return lines
}

func multipartPartContentTypeForCurl(field string, encodingByField map[string]model.MediaTypeEncoding, jsonFields map[string]bool) string {
	if encoding, ok := encodingByField[field]; ok {
		if contentType := strings.TrimSpace(encoding.ContentType); contentType != "" {
			return contentType
		}
	}
	if jsonFields[field] {
		return "application/json"
	}

	return ""
}

func readRequestBody(request *http.Request) ([]byte, error) {
	if request == nil || request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func sortedHeaderNames(headers http.Header) []string {
	if len(headers) == 0 {
		return nil
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func skipCurlHeader(operation *model.Operation, draft *model.RequestDraft, name string) bool {
	if !strings.EqualFold(name, "Content-Type") {
		return false
	}
	return effectiveBodyMediaType(operation, draft) == "multipart/form-data"
}

func renderCurlFlag(flag, value string) string {
	return "  " + flag + " " + shellQuote(value)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
