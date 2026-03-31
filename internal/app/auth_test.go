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
	if issue, ok := result.IssueForTarget(AuthFieldTarget("basic_auth", AuthFieldPassword)); !ok || issue.Message != "Password is required." {
		t.Fatalf("expected missing password issue, got %#v ok=%v", issue, ok)
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
