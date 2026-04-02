package response

import (
	"fmt"
	"slices"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/x/ansi"
)

const SectionLive = "Live"

// Data contains the render-ready state for the response pane.
type Data struct {
	LoadInFlight  bool
	Sections      []widgets.Section
	ActiveSection string
	EmptyState    string
	ContentWidth  int
}

// AvailableSections returns the visible response sections for the selected operation.
func AvailableSections(responses []model.ResponseSpec) []string {
	sections := make([]string, 0, len(responses)+1)
	sections = append(sections, SectionLive)
	for _, response := range responses {
		sections = append(sections, response.StatusCode)
	}

	return sections
}

// ResolveActiveSection returns the active response section, defaulting to the live section.
func ResolveActiveSection(current string, responses []model.ResponseSpec) string {
	return widgets.ResolveActiveSection(current, AvailableSections(responses), SectionLive)
}

// MoveActiveSection moves the active response section by the requested direction.
func MoveActiveSection(current string, direction int, responses []model.ResponseSpec) string {
	return widgets.MoveActiveSection(current, AvailableSections(responses), direction, SectionLive)
}

// BoundaryActiveSection returns the first or last available response section.
func BoundaryActiveSection(last bool, responses []model.ResponseSpec) string {
	return widgets.BoundaryActiveSection(AvailableSections(responses), last, SectionLive)
}

// Sections builds the declared response sections for the selected operation.
func Sections(responses []model.ResponseSpec, contentWidth int) []widgets.Section {
	sections := make([]widgets.Section, 0, len(responses))
	for _, response := range responses {
		sections = append(sections, widgets.Section{
			Label: response.StatusCode,
			Body:  sectionBody(response, contentWidth),
		})
	}

	return sections
}

// Render renders the response pane from its render-ready data.
func Render(data Data) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   data.Sections,
		Active:     data.ActiveSection,
		EmptyState: data.EmptyState,
	})
}

// LiveSection builds the live response section for the selected operation.
func LiveSection(response *model.HTTPResponse, selected *model.Operation, contentWidth int) widgets.Section {
	body := "No request has been sent for this operation yet."
	if response != nil && selected != nil && response.OperationKey == selected.Key {
		body = liveSectionBody(response, contentWidth)
	}

	return widgets.Section{
		Label: SectionLive,
		Body:  body,
	}
}

// ActiveSectionBody returns the body for the active response section.
func ActiveSectionBody(sections []widgets.Section, active string) string {
	if len(sections) == 0 {
		return ""
	}

	if active != "" {
		for _, section := range sections {
			if section.Label == active {
				return section.Body
			}
		}
	}

	return sections[0].Body
}

// sectionBody renders a declared response section body.
func sectionBody(response model.ResponseSpec, contentWidth int) string {
	lines := []string{
		renderMetaLine("Description", describe.NormaliseInlineText(util.FallbackText(response.Description, "None"))),
	}

	lines = append(lines, "", "Headers:")
	if len(response.Headers) == 0 {
		lines = append(lines, "- none")
	} else {
		lines = append(lines, renderDeclaredHeaders(response.Headers)...)
	}

	lines = append(lines, "", "Body:")
	mediaTypes := describe.DefaultStrings(describe.MediaTypesForContent(response.Content), "none")
	lines = append(lines, renderBodyBlock(strings.Join(mediaTypes, "\n"), contentWidth)...)

	return strings.Join(lines, "\n")
}

// liveSectionBody renders the live response section body from execution output.
func liveSectionBody(response *model.HTTPResponse, contentWidth int) string {
	if response == nil {
		return "No request has been sent for this operation yet."
	}

	status := response.Status
	if strings.TrimSpace(status) == "" {
		status = "No HTTP status"
	}

	lines := []string{
		renderMetaLine("Status", status),
		renderMetaLine("Duration", fmt.Sprintf("%s", response.Duration)),
	}
	if strings.TrimSpace(response.ContentType) != "" {
		lines = append(lines, renderMetaLine("Content type", response.ContentType))
	}
	if response.ContentLength > 0 {
		lines = append(lines, renderMetaLine("Content length", fmt.Sprintf("%d", response.ContentLength)))
	}
	if strings.TrimSpace(response.TransportError) != "" {
		lines = append(lines, widgets.ErrorTextStyle().Bold(true).Render("Transport error: "+response.TransportError))
	}

	lines = append(lines, "", "Headers:")
	if len(response.Headers) == 0 {
		lines = append(lines, "- none")
	} else {
		lines = append(lines, renderWrappedHeaders(response.Headers, contentWidth)...)
	}

	lines = append(lines, "", "Body:")
	body := strings.TrimSpace(response.PrettyBody)
	if body == "" {
		body = strings.TrimSpace(string(response.Body))
	}
	if body == "" {
		body = "<empty>"
	}
	lines = append(lines, renderBodyBlock(body, contentWidth)...)

	return strings.Join(lines, "\n")
}

// renderMetaLine renders one response metadata label/value pair.
func renderMetaLine(label, value string) string {
	return widgets.MutedTextStyle().Render(label+": ") + widgets.BodyTextStyle().Render(value)
}

// renderWrappedHeaders renders sorted response headers with wrapped values.
func renderWrappedHeaders(headers map[string][]string, contentWidth int) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	slices.Sort(names)

	lines := make([]string, 0, len(headers)*2)
	for _, name := range names {
		value := strings.Join(headers[name], ", ")
		lines = append(lines, renderWrappedHeader(name, value, contentWidth)...)
	}

	return lines
}

// renderWrappedHeader renders one response header as a wrapped block.
func renderWrappedHeader(name, value string, contentWidth int) []string {
	headerLine := widgets.MutedTextStyle().Render("- " + name + ":")
	if strings.TrimSpace(value) == "" {
		return []string{headerLine, "  " + widgets.MutedTextStyle().Render("<empty>")}
	}

	wrapWidth := max(contentWidth-4, 20)
	valueLines := strings.Split(ansi.Wordwrap(value, wrapWidth, ""), "\n")
	lines := []string{headerLine}
	for _, line := range valueLines {
		lines = append(lines, "  "+line)
	}

	return lines
}

// renderDeclaredHeaders renders declared response headers.
func renderDeclaredHeaders(headers []model.Parameter) []string {
	lines := make([]string, 0, len(headers)*2)
	for _, header := range headers {
		lines = append(lines,
			widgets.MutedTextStyle().Render("- "+header.Name+":"),
			"  "+describe.ParameterTypeHint(header),
		)
	}

	return lines
}

// renderBodyBlock renders the response body as an indented content block.
func renderBodyBlock(body string, contentWidth int) []string {
	body = widgets.NormalizeRenderedBody(body)
	bodyLines := strings.Split(body, "\n")
	lines := make([]string, 0, len(bodyLines))
	wrapWidth := max(contentWidth-2, 1)
	for _, line := range bodyLines {
		wrapped := ansi.Hardwrap(line, wrapWidth, true)
		for wrappedLine := range strings.SplitSeq(wrapped, "\n") {
			lines = append(lines, widgets.MutedTextStyle().Render("│ ")+widgets.BodyTextStyle().Render(wrappedLine))
		}
	}

	return lines
}
