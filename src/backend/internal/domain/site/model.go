package site

// Summary is the site row used by operations overview APIs.
type Summary struct {
	ID                  string  `json:"id"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Campus              string  `json:"campus"`
	CapacityKW          float64 `json:"capacityKw"`
	LoadLimitKW         float64 `json:"loadLimitKw"`
	Status              string  `json:"status"`
	ChargerCount        int     `json:"chargerCount"`
	ConnectorCount      int     `json:"connectorCount"`
	AvailableConnectors int     `json:"availableConnectors"`
	ActiveSessions      int     `json:"activeSessions"`
	FaultedChargers     int     `json:"faultedChargers"`
}

// Topology represents a site, its electrical groups, chargers, and connectors.
type Topology struct {
	Site  SiteNode   `json:"site"`
	Areas []AreaNode `json:"areas"`
}

// SiteNode is the root of a charger topology tree.
type SiteNode struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Campus      string  `json:"campus"`
	CapacityKW  float64 `json:"capacityKw"`
	LoadLimitKW float64 `json:"loadLimitKw"`
	Status      string  `json:"status"`
}

// AreaNode groups chargers by physical site area.
type AreaNode struct {
	ID          string      `json:"id"`
	Code        string      `json:"code"`
	Name        string      `json:"name"`
	LoadLimitKW float64     `json:"loadLimitKw"`
	Status      string      `json:"status"`
	Groups      []GroupNode `json:"groups"`
}

// GroupNode groups chargers by electrical capacity boundary.
type GroupNode struct {
	ID             string        `json:"id"`
	Code           string        `json:"code"`
	Name           string        `json:"name"`
	ElectricalNode string        `json:"electricalNode"`
	LoadLimitKW    float64       `json:"loadLimitKw"`
	Priority       int           `json:"priority"`
	Status         string        `json:"status"`
	Chargers       []ChargerNode `json:"chargers"`
}

// ChargerNode describes an individual charger in the topology tree.
type ChargerNode struct {
	ID                   string          `json:"id"`
	Code                 string          `json:"code"`
	Name                 string          `json:"name"`
	ChargerType          string          `json:"chargerType"`
	RatedPowerKW         float64         `json:"ratedPowerKw"`
	Status               string          `json:"status"`
	InstallationLocation string          `json:"installationLocation"`
	Connectors           []ConnectorNode `json:"connectors"`
}

// ConnectorNode describes a charger connector.
type ConnectorNode struct {
	ID         string  `json:"id"`
	Code       string  `json:"code"`
	Number     int     `json:"number"`
	MaxPowerKW float64 `json:"maxPowerKw"`
	Status     string  `json:"status"`
}
