package request

import "github.com/phergul/apiscope/internal/model"

const (
	SectionBody = "Body"
	SectionAuth = "Auth"
)

var parameterLocations = []model.ParameterLocation{
	model.ParameterLocationPath,
	model.ParameterLocationQuery,
	model.ParameterLocationHeader,
	model.ParameterLocationCookie,
}

func AvailableSections(selected *model.Operation, security *model.SecurityRequirement) []string {
	if selected == nil {
		return nil
	}

	sections := make([]string, 0, len(parameterLocations)+2)
	for _, location := range parameterLocations {
		if hasParametersInLocation(selected.Parameters, location) {
			sections = append(sections, locationSectionLabel(location))
		}
	}
	if selected.RequestBody != nil {
		sections = append(sections, SectionBody)
	}
	if security != nil && len(security.Alternatives) > 0 {
		sections = append(sections, SectionAuth)
	}

	return sections
}

func hasParametersInLocation(parameters []model.Parameter, location model.ParameterLocation) bool {
	for _, parameter := range parameters {
		if parameter.In == location {
			return true
		}
	}

	return false
}

func locationSectionLabel(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path"
	case model.ParameterLocationQuery:
		return "Query"
	case model.ParameterLocationHeader:
		return "Header"
	case model.ParameterLocationCookie:
		return "Cookie"
	default:
		return string(location)
	}
}
