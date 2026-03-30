package widgets

import "strings"

// ClippedSectionViewInput describes a section view that may need active-body clipping.
type ClippedSectionViewInput struct {
	Sections      []Section
	Active        string
	EmptyState    string
	ContentWidth  int
	ContentHeight int
	ScrollOffset  int
}

// ClippedSectionView contains the clipped section view plus scroll metadata.
type ClippedSectionView struct {
	Data            SectionViewData
	Active          string
	MaxScrollOffset int
}

// ProjectClippedSectionView clips the active section body to the requested viewport.
func ProjectClippedSectionView(input ClippedSectionViewInput) ClippedSectionView {
	active := activeSectionLabel(input.Sections, input.Active)
	data := SectionViewData{
		Sections:   append([]Section(nil), input.Sections...),
		Active:     active,
		EmptyState: input.EmptyState,
	}
	if len(data.Sections) == 0 {
		return ClippedSectionView{Data: data}
	}

	maxScrollOffset := 0
	if input.ContentHeight > 0 {
		maxScrollOffset = MaxSectionScrollOffset(input.Sections, active, input.ContentHeight)
		viewport := NewViewport(max(input.ContentWidth, 1), max(input.ContentHeight, 1))
		viewport.SetContent(activeSectionBody(input.Sections, active))
		// clamp before handing the offset to bubbles so the shared helper stays deterministic.
		viewport.SetYOffset(clampSectionScrollOffset(input.ScrollOffset, maxScrollOffset))
		clippedBody := viewport.View()

		for index := range data.Sections {
			if data.Sections[index].Label == active {
				data.Sections[index].Body = clippedBody
				break
			}
		}
	}

	return ClippedSectionView{
		Data:            data,
		Active:          active,
		MaxScrollOffset: maxScrollOffset,
	}
}

// MaxSectionScrollOffset returns the largest valid scroll offset for the active section body.
func MaxSectionScrollOffset(sections []Section, active string, visibleLines int) int {
	if len(sections) == 0 || visibleLines <= 0 {
		return 0
	}

	lines := len(splitSectionLines(activeSectionBody(sections, active)))
	if lines <= visibleLines {
		return 0
	}

	return lines - visibleLines
}

// clampSectionScrollOffset bounds a section scroll offset to the current active body.
func clampSectionScrollOffset(offset, maxOffset int) int {
	if offset < 0 {
		return 0
	}
	if offset > maxOffset {
		return maxOffset
	}

	return offset
}

// splitSectionLines keeps empty content addressable as a single display line.
func splitSectionLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	return strings.Split(text, "\n")
}
