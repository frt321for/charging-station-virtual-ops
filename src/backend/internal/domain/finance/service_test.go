package finance

import "testing"

func TestValidReviewStatus(t *testing.T) {
	if !validReviewStatus("reviewing") {
		t.Fatalf("expected reviewing to be valid")
	}
	if validReviewStatus("confirmed") {
		t.Fatalf("expected confirmed to be invalid")
	}
}

func TestBuildExport(t *testing.T) {
	export := buildExport([]ExportRow{{
		ExceptionNo:   "RE-1",
		BillNo:        "BILL-1",
		SessionNo:     "CS-1",
		ExceptionType: "amount",
		Severity:      "medium",
		Status:        "open",
		Reason:        "amount mismatch",
		TotalAmount:   12.5,
	}}, "finance")

	if export.ExportNo == "" || export.CSV == "" || len(export.Rows) != 1 {
		t.Fatalf("unexpected export: %#v", export)
	}
}
