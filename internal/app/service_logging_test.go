package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestServiceLoadSourceLogsFailure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	service := NewService(&stubLoader{err: errors.New("boom")}, nil, nil, logger)

	_, err := service.LoadSource(context.Background(), "broken.yaml")
	if err == nil {
		t.Fatal("expected load error")
	}
	if !strings.Contains(buf.String(), `"event":"load_source_failed"`) {
		t.Fatalf("expected load failure log, got %s", buf.String())
	}
}

func TestServiceExecuteCurrentLogsValidationFailure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	session := model.SessionState{
		SelectedServerURL:    "https://api.example.com",
		SelectedOperationKey: model.NewOperationKey("POST", "/pets/{petId}"),
		Spec: &model.APISpec{
			Operations: []model.Operation{
				{
					Key:    model.NewOperationKey("POST", "/pets/{petId}"),
					Method: "POST",
					Path:   "/pets/{petId}",
					Parameters: []model.Parameter{
						{Name: "petId", In: model.ParameterLocationPath, Required: true},
					},
				},
			},
		},
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}

	NewService(nil, nil, nil, logger).ExecuteCurrent(context.Background(), session)
	if !strings.Contains(buf.String(), `"event":"execute_validation_failed"`) {
		t.Fatalf("expected validation failure log, got %s", buf.String())
	}
}
