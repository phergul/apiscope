package model

import "time"

type HTTPResponse struct {
	OperationKey   OperationKey
	RequestID      uint64
	StatusCode     int
	Status         string
	Duration       time.Duration
	Headers        map[string][]string
	Body           []byte
	ContentType    string
	ContentLength  int64
	PrettyBody     string
	TransportError string
}
