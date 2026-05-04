package simulator

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ModeOnce = "once"
	ModeRun  = "run"

	LoadFlat    = "flat"
	LoadCommute = "commute"
	LoadRandom  = "random"
)

// Config controls virtual charger generation and behavior.
type Config struct {
	Mode           string
	APIBase        string
	GroupCode      string
	Prefix         string
	ChargerCount   int
	ConnectorCount int
	OnlineRate     float64
	FaultRate      float64
	LoadCurve      string
	Ticks          int
	Interval       time.Duration
}

// Validate checks simulator configuration before any API calls are made.
func (c Config) Validate() error {
	if c.Mode != ModeOnce && c.Mode != ModeRun {
		return fmt.Errorf("mode must be %s or %s", ModeOnce, ModeRun)
	}
	if strings.TrimSpace(c.APIBase) == "" {
		return errors.New("api-base is required")
	}
	if strings.TrimSpace(c.GroupCode) == "" {
		return errors.New("group-code is required")
	}
	if c.ChargerCount <= 0 {
		return errors.New("chargers must be greater than 0")
	}
	if c.ConnectorCount <= 0 {
		return errors.New("connectors must be greater than 0")
	}
	if c.OnlineRate < 0 || c.OnlineRate > 1 {
		return errors.New("online-rate must be between 0 and 1")
	}
	if c.FaultRate < 0 || c.FaultRate > 1 {
		return errors.New("fault-rate must be between 0 and 1")
	}
	if c.Ticks <= 0 {
		return errors.New("ticks must be greater than 0")
	}
	if c.Interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	switch c.LoadCurve {
	case LoadFlat, LoadCommute, LoadRandom:
		return nil
	default:
		return fmt.Errorf("load-curve must be %s, %s, or %s", LoadFlat, LoadCommute, LoadRandom)
	}
}
