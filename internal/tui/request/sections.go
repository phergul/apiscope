package request

import (
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

const (
	SectionServer = "Server"
	SectionBody   = "Body"
	SectionAuth   = "Auth"
)

var parameterLocations = []model.ParameterLocation{
	model.ParameterLocationPath,
	model.ParameterLocationQuery,
	model.ParameterLocationHeader,
	model.ParameterLocationCookie,
}

// AvailableSections returns the visible request pane sections for the selected operation.
func AvailableSections(selected *model.Operation, security *model.SecurityRequirement, servers []model.Server) []string {
	if selected == nil {
		return nil
	}

	sections := make([]string, 0, len(parameterLocations)+3)
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
	if len(servers) > 1 {
		sections = append(sections, SectionServer)
	}

	return sections
}

// ResolveActiveSection resolves the current request section against the available request sections.
func ResolveActiveSection(current string, selected *model.Operation, security *model.SecurityRequirement, servers []model.Server) string {
	return widgets.ResolveActiveSection(current, AvailableSections(selected, security, servers), "")
}

// MoveActiveSection moves the active request section by one step in the given direction.
func MoveActiveSection(current string, direction int, selected *model.Operation, security *model.SecurityRequirement, servers []model.Server) string {
	return widgets.MoveActiveSection(current, AvailableSections(selected, security, servers), direction, "")
}

// BoundaryActiveSection returns the first or last available request section.
func BoundaryActiveSection(last bool, selected *model.Operation, security *model.SecurityRequirement, servers []model.Server) string {
	return widgets.BoundaryActiveSection(AvailableSections(selected, security, servers), last, "")
}

// hasParametersInLocation reports whether the selected operation exposes parameters in the given location.
func hasParametersInLocation(parameters []model.Parameter, location model.ParameterLocation) bool {
	for _, parameter := range parameters {
		if parameter.In == location {
			return true
		}
	}

	return false
}

// locationSectionLabel maps a parameter location to the request pane section label.
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
