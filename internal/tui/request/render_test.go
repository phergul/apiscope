package request

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderShowsGroupedInputsAndAuthRows(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Path", "Query", "Body", "Auth"},
		ActiveSection: "Path",
		Rows: []Row{
			{
				Label:    "petId",
				Meta:     "required, string",
				Value:    "<unset>",
				Editable: true,
			},
			{
				Label:    "legacy",
				Meta:     "optional, content",
				Value:    "content-based input",
				Editable: false,
				Support: []SupportNote{{
					Severity: SupportSeverityUnsupported,
					Summary:  "Content-based parameter is read-only.",
					Detail:   "This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
				}},
			},
		},
		ActiveRow: 0,
	}))

	wantSnippets := []string{
		"Path  Query  Body  Auth",
		" petId (required, string) = <unset>",
		"legacy (optional, content) = content-based input [read-only]",
		"unsupported: Content-based parameter is read-only. This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request pane to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsSupportSummaryAndValidationTogether(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:         []string{"Query"},
		ActiveSection:    "Query",
		ValidationNotice: []string{"Required value missing."},
		SupportNotice: []SupportNote{{
			Severity: SupportSeverityDowngraded,
			Summary:  `Swagger collectionFormat "pipes" needs manual formatting.`,
			Detail:   "Enter the fully formatted value yourself.",
		}},
		Rows: []Row{
			{
				Label:    "tags",
				Meta:     "optional, array",
				Value:    "<unset>",
				Editable: true,
				Error:    "Required value missing.",
				Support: []SupportNote{{
					Severity: SupportSeverityDowngraded,
					Summary:  `Swagger collectionFormat "pipes" needs manual formatting.`,
					Detail:   "Enter the fully formatted value yourself.",
				}},
			},
		},
		ActiveRow: 0,
	}))

	for _, snippet := range []string{
		"Support notes:",
		`downgraded: Swagger collectionFormat "pipes" needs manual formatting. Enter the fully formatted value yourself.`,
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected mixed validation/support render to include %q, got %q", snippet, content)
		}
	}
	if strings.Contains(content, "Validation:") {
		t.Fatalf("expected validation summary to hide duplicate row error, got %q", content)
	}
}

func TestRenderShowsGroupedAuthAlternativeRows(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Auth"},
		ActiveSection: "Auth",
		Rows: []Row{
			{
				Kind:  RowKindAuthOption,
				Label: "Option 1",
				Meta:  "missing 1 field",
				Value: "bearer_auth",
			},
			{
				Kind:     RowKindAuthField,
				Label:    "bearer_auth",
				Meta:     "Bearer token",
				Value:    "token set",
				Editable: true,
			},
		},
		ActiveRow: 0,
	}))

	for _, snippet := range []string{"Option 1 (missing 1 field) = bearer_auth", "bearer_auth (Bearer token) = token set"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected grouped auth content to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsUnsupportedAuthInfoRow(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Auth"},
		ActiveSection: "Auth",
		Rows: []Row{
			{
				Kind:  RowKindAuthOption,
				Label: "Option 1",
				Meta:  "unsupported",
				Value: "digest_auth",
			},
			{
				Kind:     RowKindAuthInfo,
				Label:    "digest_auth",
				Meta:     "unsupported auth",
				Value:    `HTTP auth scheme "digest" is not supported.`,
				Editable: false,
			},
		},
	}))

	for _, snippet := range []string{"Option 1 (unsupported) = digest_auth", `digest_auth (unsupported auth) = HTTP auth scheme "digest" is not supported. [read-only]`} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected unsupported auth content to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsExplicitEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		EmptyState: "This operation does not declare request parameters, request body, or auth requirements.",
	}))

	if !strings.Contains(content, "This operation does not declare request parameters, request body, or auth requirements.") {
		t.Fatalf("expected request pane empty state, got %q", content)
	}
}

func TestRenderShowsBodyEditorState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Path", "Body", "Auth"},
		ActiveSection: "Body",
		ContentWidth:  80,
		ContentHeight: 12,
		Edit: EditView{
			Kind:      "body",
			MediaType: "application/json",
			Buffer:    "{\n  \"name\": \"fido\"\n}",
			View:      "{\n  \"name\": \"fido\"\n}",
			Title:     "Edit body",
			Context:   "Media type: application/json",
		},
	}))

	wantSnippets := []string{
		"Path  Body  Auth",
		"Edit body",
		"Media type: application/json",
		"  \"name\": \"fido\"",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected body editor state to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsInlineValidationState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:         []string{"Path", "Body"},
		ActiveSection:    "Path",
		ValidationNotice: []string{"Required value missing."},
		Rows: []Row{
			{
				Label:    "petId",
				Meta:     "required, string",
				Value:    "<unset>",
				Editable: true,
				Error:    "Required value missing.",
			},
		},
		ActiveRow: 0,
	}))

	wantSnippets := []string{"petId (required, string) = <unset>", "Required value missing."}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request validation content to include %q, got %q", snippet, content)
		}
	}
	if strings.Contains(content, "Validation:") {
		t.Fatalf("expected validation summary to hide duplicate row error, got %q", content)
	}
}

func TestRenderKeepsValidationSummaryForSectionOnlyMessage(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:         []string{"Auth"},
		ActiveSection:    "Auth",
		ValidationNotice: []string{"Choose one auth option."},
		Rows: []Row{{
			Kind:     RowKindAuthOption,
			Label:    "Option 1",
			Meta:     "missing 1 field",
			Value:    "bearer_auth",
			Editable: false,
		}},
	}))

	for _, snippet := range []string{"Validation:", "- Choose one auth option."} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected section-level validation summary %q, got %q", snippet, content)
		}
	}
}

func TestRenderEnvironmentSectionShowsWorkflowAndGroups(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{SectionEnvironment},
		ActiveSection: SectionEnvironment,
		Rows: []Row{
			{Kind: RowKindEnvironmentCurrent, Label: "Loaded environment", Meta: "session state", Value: "staging", Editable: false},
			{Kind: RowKindEnvironmentSave, Label: "Save session as", Meta: "Enter saves or updates", Value: "staging", Editable: true},
			{Kind: RowKindEnvironmentApply, Label: "staging", Meta: "loaded now, Enter reloads", Value: "server: https://staging.example.com · auth: 1 env binding", Editable: true},
			{Kind: RowKindEnvironmentBinding, Label: "api_key", Meta: "api key, env var binding", Value: "APISCOPE_API_KEY", Editable: true},
			{Kind: RowKindEnvironmentDelete, Label: "Delete saved environment", Meta: "Enter confirms delete", Value: "staging", Editable: true},
		},
		ActiveRow: 1,
	}))

	for _, snippet := range []string{
		"Enter saves, loads, or edits the selected environment row.",
		"Session",
		"Save",
		"Saved environments",
		"Env var bindings",
		"Danger zone",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected environment rendering to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderEnvironmentSectionShowsEmptySavedEnvironmentHint(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{SectionEnvironment},
		ActiveSection: SectionEnvironment,
		Rows: []Row{
			{Kind: RowKindEnvironmentCurrent, Label: "Loaded environment", Value: "Session only", Editable: false},
			{Kind: RowKindEnvironmentSave, Label: "Save session as", Value: "<new name>", Editable: true},
		},
		ActiveRow: 1,
	}))

	if !strings.Contains(content, "No saved environments yet. Save session as a name to create one.") {
		t.Fatalf("expected empty-saved-environment hint, got %q", content)
	}
}

func TestPopupWidthGrowsWithLongEditorContentThenCaps(t *testing.T) {
	t.Parallel()

	short := popupWidth(Data{
		ContentWidth: 90,
		Edit: EditView{
			Kind:    "field",
			Context: "api_key source",
			View:    "ABC",
		},
	})
	long := popupWidth(Data{
		ContentWidth: 90,
		Edit: EditView{
			Kind:    "field",
			Context: "api_key source",
			View:    strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 4),
		},
	})
	if long <= short {
		t.Fatalf("expected popup width to grow with content, short=%d long=%d", short, long)
	}
	if long > 88 {
		t.Fatalf("expected field popup width to cap at 88, got %d", long)
	}
}

func TestRenderShowsMultilineBodyPreview(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Body"},
		ActiveSection: "Body",
		ContentWidth:  40,
		Rows: []Row{
			{
				Kind:     RowKindBodyText,
				Label:    "Body",
				Value:    "{\n  \"name\": \"fido\",\n  \"age\": 4\n}",
				Editable: true,
			},
		},
		ActiveRow: 0,
	}))

	for _, snippet := range []string{"Body =", "Enter edit", "│ {", "│   \"name\": \"fido\",", "│   \"age\": 4"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected multiline body preview to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsFieldEditorAsPopupWithoutDefaultControls(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Query", "Auth"},
		ActiveSection: "Query",
		ContentWidth:  80,
		ContentHeight: 12,
		Rows: []Row{
			{Label: "limit", Meta: "optional, integer", Value: "<unset>", Editable: true},
			{Label: "offset", Meta: "optional, integer", Value: "<unset>", Editable: true},
		},
		Edit: EditView{
			Kind:    "field",
			View:    "42",
			Title:   "Edit value",
			Context: "limit (optional, integer)",
		},
	}))

	for _, snippet := range []string{"Edit value", "limit (optional, integer)", "42"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request popup to include %q, got %q", snippet, content)
		}
	}
	if strings.Contains(content, "Help - ?") {
		t.Fatalf("expected help hint to stay out of the editor popup, got %q", content)
	}
	if strings.Contains(content, "Enter save") {
		t.Fatalf("expected controls to stay hidden by default, got %q", content)
	}
}

func TestRenderShowsPopupHelpWhenEnabled(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Query"},
		ActiveSection: "Query",
		ContentWidth:  80,
		ContentHeight: 12,
		Rows: []Row{
			{Label: "limit", Meta: "optional, integer", Value: "<unset>", Editable: true},
		},
		Edit: EditView{
			Kind:    "field",
			View:    "42",
			Title:   "Edit value",
			Context: "limit (optional, integer)",
		},
	}))

	for _, snippet := range []string{"Enter save", "Esc cancel", "? toggle help"} {
		if strings.Contains(content, snippet) {
			t.Fatalf("expected help popup content to stay out of request pane rendering, got %q", content)
		}
	}
	if !strings.Contains(content, "Edit value") {
		t.Fatalf("expected main editor popup to remain visible, got %q", content)
	}
}
