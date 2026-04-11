package request

import (
	"strings"
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

func TestProjectPaneBuildsReadOnlyTemplatedServerRow(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{},
		Servers: []model.Server{{
			URL: "https://{env}.example.com",
			Variables: map[string]model.ServerVariable{
				"env": {Default: "api"},
			},
		}},
		SelectedServerURL: "https://{env}.example.com",
		ActiveSection:     SectionServer,
	})

	if projection.Data.ActiveSection != SectionServer {
		t.Fatalf("expected server section, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one templated server row, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[0].Editable {
		t.Fatalf("expected templated server row to stay read-only, got %#v", projection.Data.Rows[0])
	}
	if projection.Data.Rows[0].Meta != "templated server" {
		t.Fatalf("expected templated server meta, got %#v", projection.Data.Rows[0])
	}
}

func TestProjectPaneBuildsBodyExampleRowWhenNamedExamplesExist(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{
					MediaType: "application/json",
					Examples: map[string]model.Example{
						"a-first":  {Value: map[string]any{"name": "first"}},
						"b-second": {Value: map[string]any{"name": "second"}},
					},
				}},
			},
		},
		Draft:         &model.RequestDraft{BodyMediaType: "application/json", SelectedExamples: map[string]string{"body:application/json": "b-second"}},
		ActiveSection: SectionBody,
	})

	if len(projection.Data.Rows) != 3 {
		t.Fatalf("expected media type, example, and body rows, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[1].Kind != RowKindBodyExample {
		t.Fatalf("expected body example row, got %#v", projection.Data.Rows[1])
	}
	if projection.Data.Rows[1].Value != "b-second" || !projection.Data.Rows[1].Editable {
		t.Fatalf("expected editable selected example row, got %#v", projection.Data.Rows[1])
	}
}

func TestProjectPaneBuildsMultipartBodyFieldRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{
					MediaType: "multipart/form-data",
					Schema: &model.Schema{
						Type:     "object",
						Required: []string{"description", "file"},
						Properties: map[string]*model.Schema{
							"description": {Type: "string"},
							"file":        {Type: "string", Format: "binary"},
						},
					},
				}},
			},
		},
		Draft: &model.RequestDraft{
			BodyMediaType:  "multipart/form-data",
			FormParams:     map[string]string{"description": "avatar"},
			FormFileParams: map[string]string{"file": "/tmp/demo.txt"},
		},
		ActiveSection: SectionBody,
	})

	if len(projection.Data.Rows) != 3 {
		t.Fatalf("expected media type plus multipart field rows, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[1].Label != "description" || projection.Data.Rows[1].Value != "avatar" {
		t.Fatalf("expected multipart scalar row, got %#v", projection.Data.Rows[1])
	}
	if projection.Data.Rows[2].Label != "file" || projection.Data.Rows[2].Meta != "required, file path" {
		t.Fatalf("expected multipart file row, got %#v", projection.Data.Rows[2])
	}
}

func TestProjectPaneBuildsStructuredMultipartBodyFieldRow(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
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
		},
		Draft: &model.RequestDraft{
			BodyMediaType: "multipart/form-data",
			FormParams:    map[string]string{"metadata": "{\"region\":\"ie\"}"},
		},
		ActiveSection: SectionBody,
	})

	if len(projection.Data.Rows) != 2 {
		t.Fatalf("expected media type plus structured multipart field row, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[1].Label != "metadata" || projection.Data.Rows[1].Meta != "optional, object" {
		t.Fatalf("expected structured multipart row, got %#v", projection.Data.Rows[1])
	}
	if projection.Data.Rows[1].Value != "{\"region\":\"ie\"}" {
		t.Fatalf("expected structured multipart row value, got %#v", projection.Data.Rows[1])
	}
}

func TestProjectPaneBuildsUrlencodedBodyFieldRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{
					MediaType: "application/x-www-form-urlencoded",
					Schema: &model.Schema{
						Type:     "object",
						Required: []string{"attachment"},
						Properties: map[string]*model.Schema{
							"attachment": {Type: "string", Format: "binary"},
						},
					},
				}},
			},
		},
		Draft: &model.RequestDraft{
			BodyMediaType: "application/x-www-form-urlencoded",
			FormParams:    map[string]string{"attachment": "inline-data"},
		},
		ActiveSection: SectionBody,
	})

	if len(projection.Data.Rows) != 2 {
		t.Fatalf("expected media type plus urlencoded field row, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[1].Label != "attachment" || projection.Data.Rows[1].Meta != "required, string/binary" {
		t.Fatalf("expected scalar urlencoded field row, got %#v", projection.Data.Rows[1])
	}
	if !projection.Data.Rows[1].Editable || projection.Data.Rows[1].Value != "inline-data" {
		t.Fatalf("expected editable urlencoded field value, got %#v", projection.Data.Rows[1])
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
		nil,
		"",
		nil,
	)
	if rows[1].ID == rows[3].ID || rows[1].ValidationTarget == rows[3].ValidationTarget {
		t.Fatalf("expected duplicated scheme rows to remain distinct, got %#v", rows)
	}
}

func TestProjectPaneBuildsEnvManagedAuthRowsForLoadedEnvironment(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{},
		Security: &model.SecurityRequirement{Alternatives: []model.SecurityAlternative{{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}}}},
		SecuritySchemes: map[string]model.SecurityScheme{
			"api_key": {Name: "api_key", Type: model.SecuritySchemeTypeAPIKey, In: model.ParameterLocationHeader, ParameterName: "X-API-Key"},
		},
		AuthState: map[string]model.AuthValue{"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"}},
		Environments: []model.SavedEnvironment{{
			Name: "staging",
			AuthBindings: map[string]model.SavedAuthBinding{
				"api_key": {FieldEnvVars: map[model.AuthField]string{model.AuthFieldAPIKey: "APISCOPE_API_KEY"}},
			},
		}},
		AppliedEnvironmentName: "staging",
		ActiveSection:          SectionAuth,
	})

	if len(projection.Data.Rows) != 2 {
		t.Fatalf("expected auth option and field rows, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[1].Kind != RowKindAuthField || !projection.Data.Rows[1].Editable {
		t.Fatalf("expected env-managed auth field row to stay editable for source toggling, got %#v", projection.Data.Rows[1])
	}
	if !strings.Contains(projection.Data.Rows[1].Meta, "source: env var") {
		t.Fatalf("expected auth field meta to show env source binding, got %#v", projection.Data.Rows[1])
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
				Summary:  "Content-based parameter is read-only.",
				Detail:   "This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
			}},
			RowNotes: map[string][]SupportNote{
				"query:legacy": {{
					Severity: SupportSeverityUnsupported,
					Summary:  "Content-based parameter is read-only.",
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

func TestProjectPaneBuildsEditableFormRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Parameters: []model.Parameter{
				{
					Name:     "name",
					In:       model.ParameterLocationForm,
					Required: true,
					Schema:   &model.Schema{Type: "string"},
				},
			},
			FormBodyMediaType: "application/x-www-form-urlencoded",
		},
		Draft: &model.RequestDraft{
			FormParams: map[string]string{"name": "fido"},
		},
		ActiveSection: SectionForm,
	})

	if projection.Data.ActiveSection != SectionForm {
		t.Fatalf("expected form section, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one form row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Label != "name" {
		t.Fatalf("expected form row label name, got %q", projection.Data.Rows[0].Label)
	}
	if projection.Data.Rows[0].Value != "fido" {
		t.Fatalf("expected form row value fido, got %q", projection.Data.Rows[0].Value)
	}
	if !projection.Data.Rows[0].Editable {
		t.Fatal("expected form row to be editable")
	}
}

func TestProjectPaneIncludesEnvironmentSectionForSelectedOperation(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Key: model.NewOperationKey("GET", "/pets"),
		},
		ActiveSection: SectionEnvironment,
	})

	if projection.Data.ActiveSection != SectionEnvironment {
		t.Fatalf("expected environment section, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Sections) == 0 {
		t.Fatal("expected request sections to be projected")
	}
	found := false
	for _, section := range projection.Data.Sections {
		if section == SectionEnvironment {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected environment section in %#v", projection.Data.Sections)
	}
}

func TestProjectPaneBuildsEnvironmentRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Key: model.NewOperationKey("GET", "/pets"),
		},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
			},
		},
		SecuritySchemes: map[string]model.SecurityScheme{
			"api_key": {
				Name:          "api_key",
				Type:          model.SecuritySchemeTypeAPIKey,
				In:            model.ParameterLocationHeader,
				ParameterName: "X-API-Key",
			},
		},
		ActiveSection:          SectionEnvironment,
		AppliedEnvironmentName: "staging",
		Environments: []model.SavedEnvironment{{
			Name:              "staging",
			SelectedServerURL: "https://staging.example.com",
			AuthBindings: map[string]model.SavedAuthBinding{
				"api_key": {
					FieldEnvVars: map[model.AuthField]string{
						model.AuthFieldAPIKey: "API_KEY_ENV",
					},
				},
			},
		}},
	})

	if len(projection.Data.Rows) != 6 {
		t.Fatalf("expected current, unload, save, apply, binding, and delete rows, got %#v", projection.Data.Rows)
	}
	if projection.Data.Rows[0].Kind != RowKindEnvironmentCurrent {
		t.Fatalf("expected current-environment row first, got %#v", projection.Data.Rows[0])
	}
	if projection.Data.Rows[0].Label != "Loaded environment" {
		t.Fatalf("expected clearer loaded environment label, got %#v", projection.Data.Rows[0])
	}
	if projection.Data.Rows[1].Kind != RowKindEnvironmentUnload || !projection.Data.Rows[1].Editable {
		t.Fatalf("expected editable unload row, got %#v", projection.Data.Rows[1])
	}
	if projection.Data.Rows[2].Kind != RowKindEnvironmentSave || !projection.Data.Rows[2].Editable {
		t.Fatalf("expected editable save row, got %#v", projection.Data.Rows[2])
	}
	if projection.Data.Rows[3].Kind != RowKindEnvironmentApply || projection.Data.Rows[3].Value == "" {
		t.Fatalf("expected apply row with summary, got %#v", projection.Data.Rows[3])
	}
	if projection.Data.Rows[4].Kind != RowKindEnvironmentBinding || projection.Data.Rows[4].Value != "API_KEY_ENV" {
		t.Fatalf("expected environment binding row, got %#v", projection.Data.Rows[4])
	}
	if projection.Data.Rows[5].Kind != RowKindEnvironmentDelete {
		t.Fatalf("expected delete row last, got %#v", projection.Data.Rows[5])
	}
}

func TestProjectPaneBuildsEditableFileUploadRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Parameters: []model.Parameter{
				{
					Name:          "file",
					In:            model.ParameterLocationForm,
					FormInputKind: model.FormInputKindFile,
					Required:      true,
				},
			},
			FormBodyMediaType: "multipart/form-data",
		},
		Draft: &model.RequestDraft{
			FormFileParams: map[string]string{"file": "/tmp/demo.txt"},
		},
		ActiveSection: SectionForm,
	})

	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one file upload row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Meta != "required, file path" {
		t.Fatalf("expected file path row meta, got %q", projection.Data.Rows[0].Meta)
	}
	if projection.Data.Rows[0].Value != "/tmp/demo.txt" {
		t.Fatalf("expected file path row value, got %q", projection.Data.Rows[0].Value)
	}
}
