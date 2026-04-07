package model

import "time"

type HTTPResponse struct {
	OperationKey   OperationKey        `json:"operation_key"`
	RequestID      uint64              `json:"request_id"`
	StatusCode     int                 `json:"status_code"`
	Status         string              `json:"status,omitempty"`
	Duration       time.Duration       `json:"duration"`
	Headers        map[string][]string `json:"headers,omitempty"`
	Body           []byte              `json:"body,omitempty"`
	ContentType    string              `json:"content_type,omitempty"`
	ContentLength  int64               `json:"content_length"`
	PrettyBody     string              `json:"pretty_body,omitempty"`
	TransportError string              `json:"transport_error,omitempty"`
}
