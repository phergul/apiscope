package request

func ClampActiveRow(activeRow, rowCount int) int {
	if rowCount <= 0 || activeRow < 0 {
		return 0
	}
	if activeRow >= rowCount {
		return rowCount - 1
	}

	return activeRow
}

func MoveActiveRow(activeRow, rowCount, direction int) int {
	return ClampActiveRow(activeRow+direction, rowCount)
}

func BoundaryActiveRow(rowCount int, last bool) int {
	if rowCount <= 0 || !last {
		return 0
	}

	return rowCount - 1
}

func EnsureVisibleOffset(activeRow, scrollOffset, visible int) int {
	if visible <= 0 {
		visible = 1
	}
	if activeRow < scrollOffset {
		return activeRow
	}
	if activeRow >= scrollOffset+visible {
		return activeRow - visible + 1
	}

	return scrollOffset
}
