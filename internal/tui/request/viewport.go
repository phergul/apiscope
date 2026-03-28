package request

func VisibleData(data Data, scrollOffset, visibleLines int) Data {
	if len(data.Rows) == 0 || data.Edit.Kind == "body" {
		return data
	}
	if visibleLines <= 0 {
		visibleLines = 1
	}

	offset := scrollOffset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(data.Rows) {
		offset = len(data.Rows) - 1
	}

	end := offset + visibleLines
	if end > len(data.Rows) {
		end = len(data.Rows)
	}

	data.ActiveRow = ClampActiveRow(data.ActiveRow, len(data.Rows)) - offset
	data.Rows = data.Rows[offset:end]
	return data
}
