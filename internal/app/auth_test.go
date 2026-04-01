package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestSetAuthFieldStoresAndClearsValues(t *testing.T) {
	t.Parallel()

	session := model.SessionState{}
	scheme := model.SecurityScheme{
		Name:          "api_key",
		Type:          model.SecuritySchemeTypeAPIKey,
		In:            model.ParameterLocationHeader,
		ParameterName: "X-API-Key",
	}

	SetAuthField(&session, scheme, AuthFieldAPIKey, "secret")
	if got := session.AuthState["api_key"].APIKey; got != "secret" {
		t.Fatalf("expected api key to be stored, got %q", got)
	}

	SetAuthField(&session, scheme, AuthFieldAPIKey, "")
	if _, ok := session.AuthState["api_key"]; ok {
		t.Fatalf("expected auth state to clear empty values, got %#v", session.AuthState)
	}
}

func TestValidateAuthReportsMissingFieldsForBestAlternative(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "basic_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"basic_auth": {
			Name:   "basic_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthSchemeBasic,
		},
		"bearer_auth": {
			Name:   "bearer_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthSchemeBearer,
		},
	}
	authState := map[string]model.AuthValue{
		"basic_auth": {Type: model.AuthSchemeValueTypeBasic, Username: "alice"},
	}

	result := ValidateAuth(requirement, schemes, authState)
	if !result.HasIssues() {
		t.Fatal("expected auth validation issues")
	}
	if issue, ok := result.IssueForTarget(AuthAlternativeFieldTarget(0, "basic_auth", AuthFieldPassword)); !ok || issue.Message != "Password is required." {
		t.Fatalf("expected missing password issue, got %#v ok=%v", issue, ok)
	}
}

func TestValidateAuthPassesWhenLaterAlternativeIsReady(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"bearer_auth": {
			Name:   "bearer_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthSchemeBearer,
		},
		"api_key": {
			Name:          "api_key",
			Type:          model.SecuritySchemeTypeAPIKey,
			In:            model.ParameterLocationHeader,
			ParameterName: "X-API-Key",
		},
	}
	authState := map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}

	result := ValidateAuth(requirement, schemes, authState)
	if result.HasIssues() {
		t.Fatalf("expected later ready alternative to satisfy auth, got %#v", result.Issues)
	}
}

func TestValidateAuthReportsMissingSchemeWithAlternativeAwareTarget(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "oauth"}}},
		},
	}

	result := ValidateAuth(requirement, nil, nil)
	if !result.HasIssues() {
		t.Fatal("expected missing scheme validation issue")
	}
	if issue, ok := result.IssueForTarget(AuthAlternativeSchemeTarget(0, "oauth")); !ok || issue.Message != "Security scheme is missing from the spec." {
		t.Fatalf("expected missing scheme issue, got %#v ok=%v", issue, ok)
	}
}

func TestValidateAuthReportsUnsupportedHTTPAuthVariant(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "digest_auth"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"digest_auth": {
			Name:   "digest_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthScheme("digest"),
		},
	}

	result := ValidateAuth(requirement, schemes, nil)
	if !result.HasIssues() {
		t.Fatal("expected unsupported auth validation issue")
	}
	if issue, ok := result.IssueForTarget(AuthAlternativeSchemeTarget(0, "digest_auth")); !ok || issue.Message != `HTTP auth scheme "digest" is not supported.` {
		t.Fatalf("expected unsupported auth issue, got %#v ok=%v", issue, ok)
	}
}

func TestValidateAuthReportsUnsupportedAuthSchemeType(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "oauth"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"oauth": {
			Name: "oauth",
		},
	}

	result := ValidateAuth(requirement, schemes, nil)
	if !result.HasIssues() {
		t.Fatal("expected unsupported auth scheme type issue")
	}
	if issue, ok := result.IssueForTarget(AuthAlternativeSchemeTarget(0, "oauth")); !ok || issue.Message != "Auth scheme type is not supported." {
		t.Fatalf("expected unsupported auth scheme type issue, got %#v ok=%v", issue, ok)
	}
}

func TestValidateAuthBreaksEqualIncompleteTieBySpecOrder(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "basic_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "secondary_basic"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"basic_auth": {
			Name:   "basic_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthSchemeBasic,
		},
		"secondary_basic": {
			Name:   "secondary_basic",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthSchemeBasic,
		},
	}
	authState := map[string]model.AuthValue{
		"basic_auth":      {Type: model.AuthSchemeValueTypeBasic, Username: "alice"},
		"secondary_basic": {Type: model.AuthSchemeValueTypeBasic, Username: "bob"},
	}

	result := ValidateAuth(requirement, schemes, authState)
	if !result.HasIssues() {
		t.Fatal("expected incomplete auth validation issues")
	}
	if _, ok := result.IssueForTarget(AuthAlternativeFieldTarget(0, "basic_auth", AuthFieldPassword)); !ok {
		t.Fatalf("expected spec-order tie to choose first alternative, got %#v", result.Issues)
	}
}

func TestProjectAuthAlternativesPreservesGroupedAlternativeState(t *testing.T) {
	t.Parallel()

	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}, {Name: "digest_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "missing_scheme"}}},
		},
	}
	schemes := map[string]model.SecurityScheme{
		"api_key": {
			Name:          "api_key",
			Type:          model.SecuritySchemeTypeAPIKey,
			In:            model.ParameterLocationHeader,
			ParameterName: "X-API-Key",
		},
		"digest_auth": {
			Name:   "digest_auth",
			Type:   model.SecuritySchemeTypeHTTP,
			Scheme: model.HTTPAuthScheme("digest"),
		},
	}
	authState := map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}

	projections := ProjectAuthAlternatives(requirement, schemes, authState)
	if len(projections) != 2 {
		t.Fatalf("expected two auth alternatives, got %d", len(projections))
	}
	if projections[0].Status != AuthAlternativeStatusUnsupported {
		t.Fatalf("expected first alternative to be unsupported, got %q", projections[0].Status)
	}
	if projections[0].Schemes[0].Fields[0].Satisfied != true {
		t.Fatalf("expected api_key field to be satisfied, got %#v", projections[0].Schemes[0].Fields)
	}
	if projections[1].Status != AuthAlternativeStatusMissingScheme {
		t.Fatalf("expected second alternative to report missing scheme, got %q", projections[1].Status)
	}
}

func TestValidateExecutableRequestPassesWhenBearerAuthSatisfied(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("GET", "/me"),
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			},
		},
	}
	session := model.SessionState{
		Spec: &model.APISpec{
			SecuritySchemes: map[string]model.SecurityScheme{
				"bearer_auth": {
					Name:   "bearer_auth",
					Type:   model.SecuritySchemeTypeHTTP,
					Scheme: model.HTTPAuthSchemeBearer,
				},
			},
		},
		AuthState: map[string]model.AuthValue{
			"bearer_auth": {Type: model.AuthSchemeValueTypeBearer, BearerToken: "token-123"},
		},
	}

	result := ValidateExecutableRequest(session, operation, &model.RequestDraft{})
	if result.HasIssues() {
		t.Fatalf("expected auth validation to pass, got %#v", result.Issues)
	}
}
