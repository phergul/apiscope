package normalise

import (
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"
)

type normalisationState struct {
	warnings []model.SpecWarning
}

func Document(resolved *pipeline.ResolvedDocument) (*model.APISpec, error) {
	state := &normalisationState{}

	servers := normaliseServers(resolved.OpenAPI3Doc.Servers)
	securitySchemes := normaliseSecuritySchemes(resolved.OpenAPI3Doc.Components, state)
	security := normaliseSecurityRequirements(resolved.OpenAPI3Doc.Security)

	operations, err := normaliseOperations(resolved, state)
	if err != nil {
		return nil, err
	}

	spec := &model.APISpec{
		Title:           resolved.OpenAPI3Doc.Info.Title,
		Summary:         "",
		Description:     resolved.OpenAPI3Doc.Info.Description,
		SourceFamily:    resolved.SourceFamily,
		SourceVersion:   resolved.SourceVersion,
		Capabilities:    deriveCapabilities(resolved),
		Warnings:        state.warnings,
		Servers:         servers,
		Operations:      operations,
		SecuritySchemes: securitySchemes,
		Security:        security,
	}

	spec.Fingerprint = fingerprintForSpec(spec)

	return spec, nil
}
