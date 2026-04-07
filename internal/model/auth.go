package model

type AuthSchemeValueType string

const (
	AuthSchemeValueTypeAPIKey AuthSchemeValueType = "apiKey"
	AuthSchemeValueTypeBasic  AuthSchemeValueType = "basic"
	AuthSchemeValueTypeBearer AuthSchemeValueType = "bearer"
)

// AuthField identifies one editable credential field for a supported auth scheme.
type AuthField string

const (
	AuthFieldAPIKey      AuthField = "api_key"
	AuthFieldBearerToken AuthField = "bearer_token"
	AuthFieldUsername    AuthField = "username"
	AuthFieldPassword    AuthField = "password"
)

type AuthValue struct {
	Type        AuthSchemeValueType `json:"type"`
	APIKey      string              `json:"api_key,omitempty"`
	Username    string              `json:"username,omitempty"`
	Password    string              `json:"password,omitempty"`
	BearerToken string              `json:"bearer_token,omitempty"`
}
