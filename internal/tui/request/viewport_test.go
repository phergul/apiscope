package request

import "testing"

func TestVisibleDataSlicesRowsAndRebasesActiveRow(t *testing.T) {
	t.Parallel()

	data := VisibleData(Data{
		Rows: []Row{
			{Label: "one"},
			{Label: "two"},
			{Label: "three"},
			{Label: "four"},
		},
		ActiveRow: 3,
	}, 2, 2)

	if len(data.Rows) != 2 {
		t.Fatalf("expected 2 visible rows, got %d", len(data.Rows))
	}
	if data.Rows[0].Label != "three" || data.Rows[1].Label != "four" {
		t.Fatalf("unexpected visible rows: %+v", data.Rows)
	}
	if data.ActiveRow != 1 {
		t.Fatalf("expected active row rebased into visible slice, got %d", data.ActiveRow)
	}
}
