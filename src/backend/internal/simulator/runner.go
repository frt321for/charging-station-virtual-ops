package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type logFunc func(format string, args ...any)

// Runner executes virtual charger scenarios.
type Runner struct {
	cfg    Config
	client *Client
	rng    *rand.Rand
	logf   logFunc
	runID  string
}

// NewRunner creates a virtual charger runner.
func NewRunner(cfg Config, client *Client, logf logFunc) *Runner {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Runner{
		cfg:    cfg,
		client: client,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		logf:   logf,
		runID:  time.Now().UTC().Format("20060102150405"),
	}
}

// RunOnce performs one full reservation-to-stop scenario per generated charger.
func (r *Runner) RunOnce(ctx context.Context) ([]SessionResult, error) {
	results := make([]SessionResult, 0, r.cfg.ChargerCount)
	for index := 1; index <= r.cfg.ChargerCount; index++ {
		code := chargerCode(r.cfg.Prefix, r.runID, index)
		result, err := r.runChargerOnce(ctx, code, index)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

// RunContinuous registers virtual chargers and emits heartbeat/offline/fault events until cancelled.
func (r *Runner) RunContinuous(ctx context.Context) error {
	chargers, err := r.registerFleet(ctx)
	if err != nil {
		return err
	}
	r.logf("simulator running chargers=%d interval=%s", len(chargers), r.cfg.Interval)

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()
	for {
		if err := r.tickFleet(ctx, chargers); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) runChargerOnce(ctx context.Context, code string, index int) (SessionResult, error) {
	connector := connectorCode(code, 1)
	result := SessionResult{ChargerCode: code, ConnectorCode: connector, FinalStatus: "offline"}
	if err := r.registerCharger(ctx, code, index); err != nil {
		return result, err
	}
	if !r.happens(r.cfg.OnlineRate) {
		if err := r.client.Offline(ctx, code); err != nil {
			return result, err
		}
		r.logf("charger offline code=%s", code)
		return result, nil
	}

	if err := r.client.Heartbeat(ctx, code, heartbeatRequest{Status: "available", Payload: r.payload("heartbeat")}); err != nil {
		return result, err
	}
	session, err := r.client.Reservation(ctx, reservationRequest{
		ConnectorCode:      connector,
		ReservationMinutes: 30,
		RequestedBy:        "virtual-simulator",
		Payload:            r.payload("reservation"),
	})
	if err != nil {
		return result, err
	}
	result.SessionNo = session.SessionNo

	if err := r.reportConnector(ctx, code, connector, session.SessionNo, "plugged", "plug-in"); err != nil {
		return result, err
	}
	if err := r.acceptCommand(ctx, code, session.SessionNo, "start", nil); err != nil {
		return result, err
	}
	meterKWh, err := r.emitMeterValues(ctx, code, connector, session.SessionNo, index)
	if err != nil {
		return result, err
	}
	result.Samples = r.cfg.Ticks
	result.FinalMeterKWh = meterKWh

	if r.happens(r.cfg.FaultRate) {
		if err := r.raiseFault(ctx, code, connector, session.SessionNo); err != nil {
			return result, err
		}
		result.FaultRaised = true
	}
	if err := r.acceptCommand(ctx, code, session.SessionNo, "stop", nil); err != nil {
		return result, err
	}
	if err := r.reportConnector(ctx, code, connector, session.SessionNo, "available", "release"); err != nil {
		return result, err
	}
	result.FinalStatus = "pending_billing"
	r.logf("session complete charger=%s session=%s meter=%.2f", code, session.SessionNo, meterKWh)
	return result, nil
}

func (r *Runner) registerFleet(ctx context.Context) ([]string, error) {
	chargers := make([]string, 0, r.cfg.ChargerCount)
	for index := 1; index <= r.cfg.ChargerCount; index++ {
		code := chargerCode(r.cfg.Prefix, r.runID, index)
		if err := r.registerCharger(ctx, code, index); err != nil {
			return chargers, err
		}
		chargers = append(chargers, code)
	}
	return chargers, nil
}

func (r *Runner) tickFleet(ctx context.Context, chargers []string) error {
	for index, code := range chargers {
		if !r.happens(r.cfg.OnlineRate) {
			if err := r.client.Offline(ctx, code); err != nil {
				return err
			}
			r.logf("offline code=%s", code)
			continue
		}
		if err := r.client.Heartbeat(ctx, code, heartbeatRequest{Status: "available", Payload: r.payload("heartbeat")}); err != nil {
			return err
		}
		if r.happens(r.cfg.FaultRate) {
			if err := r.raiseFault(ctx, code, connectorCode(code, 1), ""); err != nil {
				return err
			}
		}
		r.logf("heartbeat code=%s index=%d", code, index+1)
	}
	return nil
}

func (r *Runner) registerCharger(ctx context.Context, code string, index int) error {
	req := registerRequest{
		GroupCode:            r.cfg.GroupCode,
		Code:                 code,
		Name:                 fmt.Sprintf("Virtual Charger %s", code),
		ChargerType:          "ac",
		RatedPowerKW:         float64(r.cfg.ConnectorCount) * 7,
		ConnectorCount:       r.cfg.ConnectorCount,
		ConnectorMaxPowerKW:  7,
		InstallationLocation: fmt.Sprintf("simulator/%s/%03d", r.runID, index),
		Payload:              r.payload("register"),
	}
	if err := r.client.Register(ctx, req); err != nil {
		return err
	}
	r.logf("registered code=%s connectors=%d", code, r.cfg.ConnectorCount)
	return nil
}

func (r *Runner) reportConnector(ctx context.Context, chargerCode string, connectorCode string, sessionNo string, status string, event string) error {
	chargerStatus := status
	if status == "plugged" {
		chargerStatus = "occupied"
	}
	return r.client.Status(ctx, chargerCode, statusRequest{
		Status:     chargerStatus,
		SessionNo:  sessionNo,
		Connectors: []connectorStatus{{Code: connectorCode, Status: status}},
		Payload:    r.payload(event),
	})
}

func (r *Runner) acceptCommand(ctx context.Context, chargerCode string, sessionNo string, commandType string, targetPower *float64) error {
	command, err := r.client.Command(ctx, sessionNo, commandType, commandRequest{
		RequestedBy:   "virtual-simulator",
		TargetPowerKW: targetPower,
		Payload:       r.payload("command-" + commandType),
	})
	if err != nil {
		return err
	}
	return r.client.Receipt(ctx, chargerCode, receiptRequest{
		CommandNo:   command.CommandNo,
		SessionNo:   sessionNo,
		CommandType: commandType,
		Receipt:     "accepted",
		Message:     "accepted by virtual simulator",
		Payload:     r.payload("receipt-" + commandType),
	})
}

func (r *Runner) emitMeterValues(ctx context.Context, chargerCode string, connectorCode string, sessionNo string, index int) (float64, error) {
	meterKWh := 1000 + float64(index*10)
	for tick := 0; tick < r.cfg.Ticks; tick++ {
		powerKW := powerFor(r.cfg.LoadCurve, tick, r.cfg.Ticks, 7, r.rng)
		meterKWh += powerKW * r.cfg.Interval.Hours()
		req := meterValueRequest{
			ConnectorCode: connectorCode,
			SessionNo:     sessionNo,
			Time:          time.Now().UTC().Add(time.Duration(tick) * r.cfg.Interval),
			PowerKW:       powerKW,
			VoltageV:      226,
			CurrentA:      powerKW * 1000 / 226,
			MeterKWh:      round2(meterKWh),
			Payload:       r.payload("meter"),
		}
		if err := r.client.MeterValue(ctx, chargerCode, req); err != nil {
			return meterKWh, err
		}
	}
	return round2(meterKWh), nil
}

func (r *Runner) raiseFault(ctx context.Context, chargerCode string, connectorCode string, sessionNo string) error {
	return r.client.Alarm(ctx, chargerCode, alarmRequest{
		ConnectorCode: connectorCode,
		SessionNo:     sessionNo,
		FaultCode:     "SIMULATED_FAULT",
		Severity:      "low",
		OccurredAt:    time.Now().UTC(),
		Payload:       r.payload("fault"),
	})
}

func (r *Runner) payload(event string) map[string]any {
	return map[string]any{
		"source": "virtual-simulator",
		"event":  event,
		"runId":  r.runID,
	}
}

func (r *Runner) happens(rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	return r.rng.Float64() <= rate
}
