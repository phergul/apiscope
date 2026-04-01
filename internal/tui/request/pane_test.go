package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestProjectPaneResolvesActiveSectionBeforeProjectingRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Parameters: []model.Parameter{
				{
					Name: "limit",
					In:   model.ParameterLocationQuery,
				},
			},
		},
		Draft: &model.RequestDraft{
			QueryParams: map[string]string{"limit": "10"},
		},
		ActiveSection: "Missing",
	})

	if projection.Data.ActiveSection != "Query" {
		t.Fatalf("expected active section to fall back to Query, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one projected row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Label != "limit" {
		t.Fatalf("expected projected row label limit, got %q", projection.Data.Rows[0].Label)
	}
}

func TestProjectPaneKeepsOnlyHintWhenHelpIsClosed(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
			},
		},
		Draft:         &model.RequestDraft{},
		ActiveSection: SectionBody,
		Editor: EditorInput{
			Kind:   model.RequestEditKindBody,
			Buffer: "{}",
		},
		HelpOpen: false,
	})

	if projection.HelpOverlay.Hint != "Help - ?" {
		t.Fatalf("expected closed help overlay hint, got %q", projection.HelpOverlay.Hint)
	}
	if projection.HelpOverlay.Title != "" || projection.HelpOverlay.Body != "" {
		t.Fatalf("expected closed help overlay to omit body, got %+v", projection.HelpOverlay)
	}
}

func TestProjectPaneBuildsEditableAuthRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			},
		},
		SecuritySchemes: map[string]model.SecurityScheme{
			"bearer_auth": {
				Name:   "bearer_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
		AuthState: map[string]model.AuthValue{
			"bearer_auth": {Type: model.AuthSchemeValueTypeBearer, BearerToken: "secret"},
		},
		ActiveSection: SectionAuth,
	})

	if projection.Data.ActiveSection != SectionAuth {
		t.Fatalf("expected auth section, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 2 {
		t.Fatalf("expected option header plus auth row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Kind != RowKindAuthOption {
		t.Fatalf("expected option header row first, got %#v", projection.Data.Rows[0])
	}
	if !projection.Data.Rows[1].Editable {
		t.Fatal("expected auth row to be editable")
	}
	if projection.Data.Rows[1].Value != "token set" {
		t.Fatalf("expected masked auth summary, got %q", projection.Data.Rows[1].Value)
	}
}

func TestProjectPaneBuildsServerSwitchRow(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{},
		Servers: []model.Server{
			{URL: "https://api.example.com", Description: "Production"},
			{URL: "https://staging.example.com", Description: "Staging"},
		},
		SelectedServerURL: "https://staging.example.com",
		ActiveSection:     SectionServer,
	})

	if projection.Data.ActiveSection != SectionServer {
		t.Fatalf("expected server section, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one server row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Label != "Base URL" {
		t.Fatalf("expected base url row label, got %q", projection.Data.Rows[0].Label)
	}
	if projection.Data.Rows[0].Value != "https://staging.example.com" {
		t.Fatalf("expected selected server url to render, got %q", projection.Data.Rows[0].Value)
	}
}

func TestProjectPaneBuildsAlternativeBlocksWithoutDeduplicatingSchemes(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			},
		},
		SecuritySchemes: map[string]model.SecurityScheme{
			"bearer_auth": {
				Name:   "bearer_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
		ActiveSection: SectionAuth,
	})

	if len(projection.Data.Rows) != 4 {
		t.Fatalf("expected two option blocks with one field each, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[1].Kind != RowKindAuthField || projection.Data.Rows[3].Kind != RowKindAuthField {
		t.Fatalf("expected auth field rows under each option, got %#v", projection.Data.Rows)
	}
	rows := ActiveRows(
		&model.Operation{},
		nil,
		SectionAuth,
		&model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			},
		},
		nil,
		"",
		map[string]model.SecurityScheme{
			"bearer_auth": {
				Name:   "bearer_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
		nil,
	)
	if rows[1].ID == rows[3].ID || rows[1].ValidationTarget == rows[3].ValidationTarget {
		t.Fatalf("expected duplicated scheme rows to remain distinct, got %#v", rows)
	}
}

func TestProjectPaneBuildsSupportNotesForActiveSectionAndRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Parameters: []model.Parameter{
				{
					Name:    "legacy",
					In:      model.ParameterLocationQuery,
					Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
				},
			},
		},
		Draft:         &model.RequestDraft{},
		ActiveSection: "Query",
		Support: SupportState{
			MessagesBySection: []SupportNote{{
				Severity: SupportSeverityUnsupported,
				Summary:  "Content-based parameter is read-only in v1.",
				Detail:   "This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
			}},
			RowNotes: map[string][]SupportNote{
				"query:legacy": {{
					Severity: SupportSeverityUnsupported,
					Summary:  "Content-based parameter is read-only in v1.",
					Detail:   "This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
				}},
			},
		},
	})

	if len(projection.Data.SupportNotice) != 1 {
		t.Fatalf("expected section support note, got %#v", projection.Data.SupportNotice)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one projected row, got %d", len(projection.Data.Rows))
	}
	if len(projection.Data.Rows[0].Support) != 1 {
		t.Fatalf("expected row support note, got %#v", projection.Data.Rows[0].Support)
	}
	if projection.Data.Rows[0].Value != "content-based input" {
		t.Fatalf("expected clearer content-based row value, got %q", projection.Data.Rows[0].Value)
	}
}
