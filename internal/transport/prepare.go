package transport

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/logging"
	"github.com/phergul/apiscope/internal/model"
)

type Executor struct {
	client *http.Client
	logger *slog.Logger
}

// NewExecutor builds a transport executor with the provided HTTP client.
func NewExecutor(client *http.Client, logger *slog.Logger) *Executor {
	if client == nil {
		client = http.DefaultClient
	}

	return &Executor{
		client: client,
		logger: logging.OrNop(logger).With("component", "transport"),
	}
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

	body, contentType, err := prepareRequestBody(operation, draft)
	if err != nil {
		return nil, err
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
		if strings.TrimSpace(contentType) != "" {
			request.Header.Set("Content-Type", contentType)
		}
	}
	if err := applyAuth(request, requirement, securitySchemes, authState); err != nil {
		return nil, err
	}

	return request, nil
}

// prepareRequestBody builds the outbound request body for the active operation.
func prepareRequestBody(operation *model.Operation, draft *model.RequestDraft) (io.Reader, string, error) {
	if operation == nil {
		return nil, "", nil
	}

	switch effectiveBodyMediaType(operation, draft) {
	case "application/x-www-form-urlencoded":
		return strings.NewReader(encodeFormParams(draft)), "application/x-www-form-urlencoded", nil
	case "multipart/form-data":
		return encodeMultipartForm(operation, draft)
	default:
		if draft != nil && draft.BodyRaw != "" {
			return strings.NewReader(draft.BodyRaw), strings.TrimSpace(draft.BodyMediaType), nil
		}
		return nil, "", nil
	}
}

func effectiveBodyMediaType(operation *model.Operation, draft *model.RequestDraft) string {
	if draft != nil && strings.TrimSpace(draft.BodyMediaType) != "" {
		return strings.TrimSpace(draft.BodyMediaType)
	}

	return strings.TrimSpace(operation.FormBodyMediaType)
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

// encodeMultipartForm serializes scalar, structured, and file form inputs into a multipart request body.
func encodeMultipartForm(operation *model.Operation, draft *model.RequestDraft) (io.Reader, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	jsonFields := multipartJSONFields(operation, draft)

	for _, key := range sortedKeys(draftFormValues(draft)) {
		value := strings.TrimSpace(draft.FormParams[key])
		if value == "" {
			continue
		}
		if jsonFields[key] {
			if err := writeMultipartJSONField(writer, key, value); err != nil {
				return nil, "", err
			}
			continue
		}
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", err
		}
	}

	for _, key := range sortedKeys(draftFileValues(draft)) {
		path := strings.TrimSpace(draft.FormFileParams[key])
		if path == "" {
			continue
		}
		if err := writeMultipartFile(writer, key, path); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &body, writer.FormDataContentType(), nil
}

func writeMultipartJSONField(writer *multipart.Writer, fieldName, value string) error {
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": fieldName}))
	headers.Set("Content-Type", "application/json")

	part, err := writer.CreatePart(headers)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, strings.NewReader(value))
	return err
}

func writeMultipartFile(writer *multipart.Writer, fieldName, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.New(`file form parameter "` + fieldName + `" path "` + path + `": ` + err.Error())
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return errors.New(`file form parameter "` + fieldName + `" path "` + path + `": ` + err.Error())
	}
	if info.IsDir() {
		return errors.New(`file form parameter "` + fieldName + `" path "` + path + `": must be a file`)
	}

	part, err := writer.CreateFormFile(fieldName, filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return errors.New(`file form parameter "` + fieldName + `" path "` + path + `": ` + err.Error())
	}

	return nil
}

func draftFormValues(draft *model.RequestDraft) map[string]string {
	if draft == nil {
		return nil
	}

	return draft.FormParams
}

func draftFileValues(draft *model.RequestDraft) map[string]string {
	if draft == nil {
		return nil
	}

	return draft.FormFileParams
}

func multipartJSONFields(operation *model.Operation, draft *model.RequestDraft) map[string]bool {
	if operation == nil || operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return nil
	}
	if effectiveBodyMediaType(operation, draft) != "multipart/form-data" {
		return nil
	}

	schema := multipartBodySchema(operation, draft)
	if schema == nil || len(schema.Properties) == 0 {
		return nil
	}

	fields := make(map[string]bool)
	for name, property := range schema.Properties {
		if multipartUsesJSONPart(property) {
			fields[name] = true
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func multipartBodySchema(operation *model.Operation, draft *model.RequestDraft) *model.Schema {
	mediaType := effectiveBodyMediaType(operation, draft)
	for _, content := range operation.RequestBody.Content {
		if content.MediaType == mediaType {
			return content.Schema
		}
	}
	return nil
}

func multipartUsesJSONPart(schema *model.Schema) bool {
	if schema == nil {
		return false
	}
	if strings.TrimSpace(schema.Type) == "array" || strings.TrimSpace(schema.Type) == "object" {
		return true
	}
	if len(schema.Properties) > 0 || schema.Items != nil || len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 || len(schema.AllOf) > 0 {
		return true
	}
	return false
}

func sortedKeys(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
