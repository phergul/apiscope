package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/phergul/apiscope/internal/model"
)

func (e *Executor) Execute(ctx context.Context, operationKey model.OperationKey, request *http.Request) *model.HTTPResponse {
	response := &model.HTTPResponse{
		OperationKey: operationKey,
	}

	if request == nil {
		response.TransportError = "request was not prepared"
		return response
	}

	start := time.Now()
	httpRequest := request.Clone(ctx)
	httpResponse, err := e.client.Do(httpRequest)
	if err != nil {
		response.Duration = time.Since(start)
		response.TransportError = err.Error()
		return response
	}
	defer httpResponse.Body.Close()

	body, readErr := io.ReadAll(httpResponse.Body)
	response.Duration = time.Since(start)
	response.StatusCode = httpResponse.StatusCode
	response.Status = httpResponse.Status
	response.Headers = httpResponse.Header.Clone()
	response.ContentType = normaliseContentType(httpResponse.Header.Get("Content-Type"))
	response.ContentLength = httpResponse.ContentLength
	if response.ContentLength < 0 {
		response.ContentLength = int64(len(body))
	}
	response.Body = body
	response.PrettyBody = prettyBody(response.ContentType, body)
	if readErr != nil {
		response.TransportError = readErr.Error()
	}

	return response
}

func normaliseContentType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return value
	}

	return mediaType
}

func prettyBody(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}

	if strings.Contains(contentType, "json") {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, body, "", "  "); err == nil {
			return formatted.String()
		}
	}

	return string(body)
}
