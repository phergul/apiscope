package tui

type layoutHeightPreset string

const (
	layoutHeightPresetCompact layoutHeightPreset = "compact"
	layoutHeightPresetNormal  layoutHeightPreset = "normal"
	layoutHeightPresetRoomy   layoutHeightPreset = "roomy"
)

type stackedPaneHeights struct {
	Operations int
	Details    int
	Expanded   int
	Folded     int
}

func chooseLayoutPreset(width int) string {
	if width >= 100 {
		return layoutPresetWide
	}

	return layoutPresetNarrow
}

func chooseHeightPreset(bodyHeight int) layoutHeightPreset {
	switch {
	case bodyHeight >= 28:
		return layoutHeightPresetRoomy
	case bodyHeight >= 20:
		return layoutHeightPresetNormal
	default:
		return layoutHeightPresetCompact
	}
}

func computeWidePaneHeights(total int) stackedPaneHeights {
	var detailsTarget, foldedTarget int
	switch chooseHeightPreset(total) {
	case layoutHeightPresetRoomy:
		detailsTarget, foldedTarget = 9, 6
	case layoutHeightPresetNormal:
		detailsTarget, foldedTarget = 7, 5
	default:
		detailsTarget, foldedTarget = 5, 4
	}

	fixedHeights, expanded := allocateExpandedStackHeights(total, []int{detailsTarget, foldedTarget}, []int{4, 0}, 6, []int{1, 0})
	return stackedPaneHeights{
		Details:  fixedHeights[0],
		Expanded: expanded,
		Folded:   fixedHeights[1],
	}
}

func computeNarrowPaneHeights(total int) stackedPaneHeights {
	var operationsTarget, detailsTarget, foldedTarget int
	switch chooseHeightPreset(total) {
	case layoutHeightPresetRoomy:
		operationsTarget, detailsTarget, foldedTarget = 10, 8, 6
	case layoutHeightPresetNormal:
		operationsTarget, detailsTarget, foldedTarget = 8, 6, 5
	default:
		operationsTarget, detailsTarget, foldedTarget = 6, 5, 4
	}

	fixedHeights, expanded := allocateExpandedStackHeights(total, []int{operationsTarget, detailsTarget, foldedTarget}, []int{4, 4, 0}, 6, []int{2, 1, 0})
	return stackedPaneHeights{
		Operations: fixedHeights[0],
		Details:    fixedHeights[1],
		Expanded:   expanded,
		Folded:     fixedHeights[2],
	}
}

func allocateExpandedStackHeights(total int, fixedTargets, fixedMinimums []int, expandedMinimum int, compressionOrder []int) ([]int, int) {
	fixedHeights := append([]int(nil), fixedTargets...)
	fixedTotal := 0
	for _, height := range fixedHeights {
		fixedTotal += height
	}
	expanded := total - fixedTotal
	if expanded >= expandedMinimum {
		return fixedHeights, expanded
	}

	deficit := expandedMinimum - expanded
	for _, index := range compressionOrder {
		if index < 0 || index >= len(fixedHeights) || index >= len(fixedMinimums) {
			continue
		}

		reducible := fixedHeights[index] - fixedMinimums[index]
		if reducible <= 0 {
			continue
		}

		delta := min(deficit, reducible)
		fixedHeights[index] -= delta
		deficit -= delta
		if deficit == 0 {
			break
		}
	}

	fixedTotal = 0
	for _, height := range fixedHeights {
		fixedTotal += height
	}

	return fixedHeights, total - fixedTotal
}
