package model

import "time"

type HTTPResponse struct {
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
