package finance

import "time"

// Correction records a finance correction without modifying raw meter readings.
type Correction struct {
	ID                       string    `json:"id"`
	CorrectionNo             string    `json:"correctionNo"`
	BillID                   string    `json:"billId"`
	BillNo                   string    `json:"billNo"`
	ExceptionID              *string   `json:"exceptionId,omitempty"`
	ExceptionNo              string    `json:"exceptionNo"`
	CorrectedEnergyKWh       *float64  `json:"correctedEnergyKwh,omitempty"`
	CorrectedDurationMinutes *int      `json:"correctedDurationMinutes,omitempty"`
	CorrectedTotalAmount     *float64  `json:"correctedTotalAmount,omitempty"`
	Reason                   string    `json:"reason"`
	ReviewerName             string    `json:"reviewerName"`
	Status                   string    `json:"status"`
	CreatedAt                time.Time `json:"createdAt"`
}

// ReviewParams updates exception review state.
type ReviewParams struct {
	ExceptionID  string
	Status       string
	ReviewerName string
	Note         string
}

// CorrectionParams creates a correction record for an exception.
type CorrectionParams struct {
	ExceptionID              string
	CorrectedEnergyKWh       *float64
	CorrectedDurationMinutes *int
	CorrectedTotalAmount     *float64
	Reason                   string
	ReviewerName             string
}

// ConfirmParams confirms a bill and closes linked reconciliation exceptions.
type ConfirmParams struct {
	BillID       string
	ReviewerName string
	Note         string
}

// ReconciliationExport is a finance export package.
type ReconciliationExport struct {
	ExportNo    string      `json:"exportNo"`
	Filename    string      `json:"filename"`
	GeneratedBy string      `json:"generatedBy"`
	GeneratedAt time.Time   `json:"generatedAt"`
	Rows        []ExportRow `json:"rows"`
	CSV         string      `json:"csv"`
}

// ExportRow is one exported reconciliation row.
type ExportRow struct {
	ExceptionNo   string  `json:"exceptionNo"`
	BillNo        string  `json:"billNo"`
	SessionNo     string  `json:"sessionNo"`
	ExceptionType string  `json:"exceptionType"`
	Severity      string  `json:"severity"`
	Status        string  `json:"status"`
	Reason        string  `json:"reason"`
	TotalAmount   float64 `json:"totalAmount"`
}
