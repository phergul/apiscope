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

	"github.com/phergul/apiscope/internal/logging"
	"github.com/phergul/apiscope/internal/model"
)

// Execute sends the prepared HTTP request and captures a normalized response.
func (e *Executor) Execute(ctx context.Context, operationKey model.OperationKey, request *http.Request) *model.HTTPResponse {
	response := &model.HTTPResponse{
		OperationKey: operationKey,
	}

	if request == nil {
		response.TransportError = "request was not prepared"
		e.logger.Error(
			"request execution skipped",
			"event", "execute_skipped",
			"operation_key", operationKey,
			"error", response.TransportError,
		)
		return response
	}

	start := time.Now()
	httpRequest := request.Clone(ctx)
	e.logger.Info(
		"sending request",
		"event", "execute_start",
		"operation_key", operationKey,
		"method", httpRequest.Method,
		"url", logging.SafeURL(httpRequest.URL.String()),
		"query_keys", logging.QueryKeys(httpRequest.URL.String()),
		"header_names", logging.HeaderNames(httpRequest.Header),
	)
	httpResponse, err := e.client.Do(httpRequest)
	if err != nil {
		response.Duration = time.Since(start)
		response.TransportError = err.Error()
		e.logger.Error(
			"request execution failed",
			"event", "execute_failed",
			"operation_key", operationKey,
			"method", httpRequest.Method,
			"url", logging.SafeURL(httpRequest.URL.String()),
			"duration_ms", response.Duration.Milliseconds(),
			"error", response.TransportError,
		)
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
		e.logger.Error(
			"response body read failed",
			"event", "response_read_failed",
			"operation_key", operationKey,
			"method", httpRequest.Method,
			"url", logging.SafeURL(httpRequest.URL.String()),
			"status_code", response.StatusCode,
			"duration_ms", response.Duration.Milliseconds(),
			"error", response.TransportError,
		)
		return response
	}

	e.logger.Info(
		"request execution completed",
		"event", "execute_complete",
		"operation_key", operationKey,
		"method", httpRequest.Method,
		"url", logging.SafeURL(httpRequest.URL.String()),
		"status_code", response.StatusCode,
		"duration_ms", response.Duration.Milliseconds(),
		"content_type", response.ContentType,
		"content_length", response.ContentLength,
		"header_names", logging.HeaderNames(response.Headers),
	)

	return response
}

// normaliseContentType strips parameters from a content type value.
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

// prettyBody pretty-prints response bytes when the content type is JSON.
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
