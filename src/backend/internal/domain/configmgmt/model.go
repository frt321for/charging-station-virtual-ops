package configmgmt

import "time"

// Snapshot is the editable configuration surface for one station.
type Snapshot struct {
	Site             SiteConfig        `json:"site"`
	Areas            []AreaConfig      `json:"areas"`
	Groups           []GroupConfig     `json:"groups"`
	Chargers         []ChargerConfig   `json:"chargers"`
	Connectors       []ConnectorConfig `json:"connectors"`
	PricingPolicies  []PricingPolicy   `json:"pricingPolicies"`
	LoadPolicies     []LoadPolicy      `json:"loadPolicies"`
	ReservationRules []ReservationRule `json:"reservationRules"`
	QueueRules       []QueueRule       `json:"queueRules"`
}

// SiteConfig describes station-wide editable operating boundaries.
type SiteConfig struct {
	ID                 string    `json:"id"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	Campus             string    `json:"campus"`
	CapacityKW         float64   `json:"capacityKw"`
	LoadLimitKW        float64   `json:"loadLimitKw"`
	Timezone           string    `json:"timezone"`
	SLAResponseMinutes int       `json:"slaResponseMinutes"`
	SLARecoveryMinutes int       `json:"slaRecoveryMinutes"`
	Status             string    `json:"status"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type AreaConfig struct {
	ID          string    `json:"id"`
	SiteID      string    `json:"siteId"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	LoadLimitKW float64   `json:"loadLimitKw"`
	SortOrder   int       `json:"sortOrder"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type GroupConfig struct {
	ID             string    `json:"id"`
	SiteID         string    `json:"siteId"`
	AreaID         string    `json:"areaId"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	ElectricalNode string    `json:"electricalNode"`
	LoadLimitKW    float64   `json:"loadLimitKw"`
	Priority       int       `json:"priority"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ChargerConfig struct {
	ID                   string    `json:"id"`
	GroupID              string    `json:"groupId"`
	GroupCode            string    `json:"groupCode"`
	Code                 string    `json:"code"`
	Name                 string    `json:"name"`
	ChargerType          string    `json:"chargerType"`
	RatedPowerKW         float64   `json:"ratedPowerKw"`
	ConnectorCount       int       `json:"connectorCount"`
	Status               string    `json:"status"`
	InstallationLocation string    `json:"installationLocation"`
	MaintenanceTag       string    `json:"maintenanceTag"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type ConnectorConfig struct {
	ID          string    `json:"id"`
	ChargerID   string    `json:"chargerId"`
	ChargerCode string    `json:"chargerCode"`
	Code        string    `json:"code"`
	Number      int       `json:"number"`
	MaxPowerKW  float64   `json:"maxPowerKw"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

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

type PricingPeriod struct {
	ID                    string  `json:"id"`
	Label                 string  `json:"label"`
	StartMinute           int     `json:"startMinute"`
	EndMinute             int     `json:"endMinute"`
	EnergyPricePerKWh     float64 `json:"energyPricePerKwh"`
	ServiceFeePerKWh      float64 `json:"serviceFeePerKwh"`
	OccupancyFeePerMinute float64 `json:"occupancyFeePerMinute"`
}

type LoadPolicy struct {
	ID          string    `json:"id"`
	SiteID      string    `json:"siteId"`
	ScopeType   string    `json:"scopeType"`
	ScopeCode   string    `json:"scopeCode"`
	ScopeName   string    `json:"scopeName"`
	ThresholdKW float64   `json:"thresholdKw"`
	WarningKW   float64   `json:"warningKw"`
	ActionMode  string    `json:"actionMode"`
	Version     int       `json:"version"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ReservationRule struct {
	ID            string    `json:"id"`
	SiteID        string    `json:"siteId"`
	SiteCode      string    `json:"siteCode"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	HoldMinutes   int       `json:"holdMinutes"`
	TimeoutAction string    `json:"timeoutAction"`
	Version       int       `json:"version"`
	Status        string    `json:"status"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type QueueRule struct {
	ID             string    `json:"id"`
	SiteID         string    `json:"siteId"`
	SiteCode       string    `json:"siteCode"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	Strategy       string    `json:"strategy"`
	MaxQueueSize   int       `json:"maxQueueSize"`
	PriorityFactor string    `json:"priorityFactor"`
	Version        int       `json:"version"`
	Status         string    `json:"status"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Boundary struct {
	SiteID   string
	SiteCode string
}

type SiteUpdate struct {
	Name               *string
	Campus             *string
	CapacityKW         *float64
	LoadLimitKW        *float64
	Timezone           *string
	SLAResponseMinutes *int
	SLARecoveryMinutes *int
	Status             *string
}

type AreaUpdate struct {
	Name        *string
	LoadLimitKW *float64
	SortOrder   *int
	Status      *string
}

type GroupUpdate struct {
	Name           *string
	ElectricalNode *string
	LoadLimitKW    *float64
	Priority       *int
	Status         *string
}

type ChargerUpdate struct {
	Name                 *string
	RatedPowerKW         *float64
	Status               *string
	InstallationLocation *string
	MaintenanceTag       *string
}

type ConnectorUpdate struct {
	MaxPowerKW *float64
	Status     *string
}

type LoadPolicyUpdate struct {
	ThresholdKW *float64
	WarningKW   *float64
	ActionMode  *string
	Status      *string
}

type ReservationRuleUpdate struct {
	Name          *string
	HoldMinutes   *int
	TimeoutAction *string
	Status        *string
}

type QueueRuleUpdate struct {
	Name           *string
	Strategy       *string
	MaxQueueSize   *int
	PriorityFactor *string
	Status         *string
}

type PricingPolicyCreate struct {
	SiteID      string
	Code        string
	Name        string
	ChargerType string
	Status      string
	Periods     []PricingPeriodInput
}

type PricingPeriodInput struct {
	Label                 string  `json:"label"`
	StartMinute           int     `json:"startMinute"`
	EndMinute             int     `json:"endMinute"`
	EnergyPricePerKWh     float64 `json:"energyPricePerKwh"`
	ServiceFeePerKWh      float64 `json:"serviceFeePerKwh"`
	OccupancyFeePerMinute float64 `json:"occupancyFeePerMinute"`
}
