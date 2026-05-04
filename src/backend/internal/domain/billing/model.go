package billing

import "time"

// PricingPolicy describes a versioned pricing rule for a site and charger type.
type PricingPolicy struct {
	ID            string          `json:"id"`
	SiteID        string          `json:"siteId"`
	SiteCode      string          `json:"siteCode"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Version       int             `json:"version"`
	ChargerType   string          `json:"chargerType"`
	EffectiveFrom time.Time       `json:"effectiveFrom"`
	EffectiveTo   *time.Time      `json:"effectiveTo,omitempty"`
	Status        string          `json:"status"`
	Periods       []PricingPeriod `json:"periods"`
}

// PricingPeriod is a daily time band inside a pricing policy.
type PricingPeriod struct {
	ID                    string  `json:"id"`
	Label                 string  `json:"label"`
	StartMinute           int     `json:"startMinute"`
	EndMinute             int     `json:"endMinute"`
	EnergyPricePerKWh     float64 `json:"energyPricePerKwh"`
	ServiceFeePerKWh      float64 `json:"serviceFeePerKwh"`
	OccupancyFeePerMinute float64 `json:"occupancyFeePerMinute"`
}

// BillingDraft is a generated bill candidate for a stopped session.
type BillingDraft struct {
	ID              string    `json:"id"`
	BillNo          string    `json:"billNo"`
	SessionID       string    `json:"sessionId"`
	SessionNo       string    `json:"sessionNo"`
	SiteID          string    `json:"siteId"`
	SiteCode        string    `json:"siteCode"`
	ConnectorCode   string    `json:"connectorCode"`
	PricingPolicyID string    `json:"pricingPolicyId"`
	PolicyCode      string    `json:"policyCode"`
	PolicyVersion   int       `json:"policyVersion"`
	EnergyKWh       float64   `json:"energyKwh"`
	DurationMinutes int       `json:"durationMinutes"`
	EnergyAmount    float64   `json:"energyAmount"`
	ServiceAmount   float64   `json:"serviceAmount"`
	OccupancyAmount float64   `json:"occupancyAmount"`
	TotalAmount     float64   `json:"totalAmount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	ExceptionFlag   bool      `json:"exceptionFlag"`
	GeneratedBy     string    `json:"generatedBy"`
	GeneratedAt     time.Time `json:"generatedAt"`
}

// ReconciliationException flags an abnormal bill or session for manual review.
type ReconciliationException struct {
	ID              string     `json:"id"`
	ExceptionNo     string     `json:"exceptionNo"`
	BillID          string     `json:"billId"`
	BillNo          string     `json:"billNo"`
	SessionID       string     `json:"sessionId"`
	SessionNo       string     `json:"sessionNo"`
	ExceptionType   string     `json:"exceptionType"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`
	Reason          string     `json:"reason"`
	SuggestedAction string     `json:"suggestedAction"`
	DetectedAt      time.Time  `json:"detectedAt"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`
}

// GenerateDraftParams describes a billing generation request.
type GenerateDraftParams struct {
	SessionID   string
	GeneratedBy string
}

type sessionForDraft struct {
	SessionID        string
	SessionNo        string
	SiteID           string
	SiteCode         string
	ConnectorCode    string
	ChargerType      string
	Status           string
	StartedAt        *time.Time
	StoppedAt        *time.Time
	StopReason       string
	MeterStartKWh    *float64
	MeterStopKWh     *float64
	FirstMeterKWh    *float64
	LastMeterKWh     *float64
	MeterSampleCount int
}

type draftCalculation struct {
	Session         sessionForDraft
	Policy          PricingPolicy
	EnergyKWh       float64
	DurationMinutes int
	EnergyAmount    float64
	ServiceAmount   float64
	OccupancyAmount float64
	TotalAmount     float64
	ExceptionFlag   bool
	Exceptions      []exceptionDraft
}

type exceptionDraft struct {
	Type            string
	Severity        string
	Reason          string
	SuggestedAction string
}
