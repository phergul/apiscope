package model

type AuthSchemeValueType string

const (
	AuthSchemeValueTypeAPIKey AuthSchemeValueType = "apiKey"
	AuthSchemeValueTypeBasic  AuthSchemeValueType = "basic"
	AuthSchemeValueTypeBearer AuthSchemeValueType = "bearer"
)

type AuthValue struct {
	Type        AuthSchemeValueType
	APIKey      string
	Username    string
	Password    string
	BearerToken string
}
