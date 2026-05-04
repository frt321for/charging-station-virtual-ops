package simcontrol

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	coresim "charging-ops/backend/internal/simulator"
)

const (
	defaultGroupCode      = "N-A1"
	defaultPrefix         = "SIM"
	defaultChargerCount   = 5
	defaultConnectorCount = 1
	defaultOnlineRate     = 0.8
	defaultFaultRate      = 0.05
	defaultLoadCurve      = coresim.LoadCommute
	defaultTicks          = 5
	defaultInterval       = 5 * time.Second
)

// Service coordinates simulator runs requested from the operations console.
type Service struct {
	apiBase string
	logger  *slog.Logger

	mu            sync.Mutex
	cancel        context.CancelFunc
	status        Status
	activeRequest Request
}

// NewService creates a simulator control service.
func NewService(apiBase string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		apiBase: apiBase,
		logger:  logger,
		status: Status{
			Mode:    "idle",
			Config:  defaultRequest(),
			Results: []coresim.SessionResult{},
		},
	}
}

// Status returns a snapshot of the simulator controller.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// RunOnce performs one virtual charger lifecycle run.
func (s *Service) RunOnce(ctx context.Context, request Request) (Status, error) {
	normalized := normalizeRequest(request)
	cfg := s.configFor(normalized, coresim.ModeOnce)
	if err := cfg.Validate(); err != nil {
		return Status{}, err
	}

	client, err := coresim.NewClient(s.apiBase, 15*time.Second)
	if err != nil {
		return Status{}, err
	}

	started := time.Now().UTC()
	s.mu.Lock()
	s.status.Mode = coresim.ModeOnce
	s.status.Config = normalized
	s.status.LastRunAt = &started
	s.status.LastError = ""
	s.status.Results = []coresim.SessionResult{}
	s.status.Generated = 0
	s.mu.Unlock()

	runner := coresim.NewRunner(cfg, client, s.logf)
	results, err := runner.RunOnce(ctx)
	completed := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.LastRunAt = &completed
	s.status.Results = results
	s.status.Generated = len(results)
	if err != nil {
		s.status.LastError = err.Error()
		return s.status, err
	}
	s.status.LastError = ""
	return s.status, nil
}

// Start begins a continuous heartbeat/fault simulator loop.
func (s *Service) Start(request Request) (Status, error) {
	normalized := normalizeRequest(request)
	cfg := s.configFor(normalized, coresim.ModeRun)
	if err := cfg.Validate(); err != nil {
		return Status{}, err
	}

	client, err := coresim.NewClient(s.apiBase, 15*time.Second)
	if err != nil {
		return Status{}, err
	}

	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return Status{}, fmt.Errorf("simulator already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	started := time.Now().UTC()
	s.cancel = cancel
	s.activeRequest = normalized
	s.status = Status{
		Running:   true,
		Mode:      coresim.ModeRun,
		Config:    normalized,
		StartedAt: &started,
		Results:   []coresim.SessionResult{},
	}
	s.mu.Unlock()

	runner := coresim.NewRunner(cfg, client, s.logf)
	go s.runContinuous(ctx, runner)

	return s.Status(), nil
}

// Stop cancels a continuous simulator loop.
func (s *Service) Stop() Status {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	stopped := time.Now().UTC()
	s.status.Running = false
	s.status.LastStoppedAt = &stopped
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return s.Status()
}

func (s *Service) runContinuous(ctx context.Context, runner *coresim.Runner) {
	err := runner.RunContinuous(ctx)
	if err != nil && ctx.Err() == nil {
		s.mu.Lock()
		s.status.LastError = err.Error()
		s.status.Running = false
		s.cancel = nil
		stopped := time.Now().UTC()
		s.status.LastStoppedAt = &stopped
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil && ctx.Err() != nil {
		s.status.Running = false
		s.cancel = nil
		stopped := time.Now().UTC()
		s.status.LastStoppedAt = &stopped
	}
}

func (s *Service) configFor(request Request, mode string) coresim.Config {
	return coresim.Config{
		Mode:           mode,
		APIBase:        s.apiBase,
		GroupCode:      defaultGroupCode,
		Prefix:         defaultPrefix,
		ChargerCount:   request.ChargerCount,
		ConnectorCount: request.ConnectorCount,
		OnlineRate:     request.OnlineRate,
		FaultRate:      request.FaultRate,
		LoadCurve:      request.LoadCurve,
		Ticks:          defaultTicks,
		Interval:       defaultInterval,
	}
}

func (s *Service) logf(format string, args ...any) {
	s.logger.Info(fmt.Sprintf(format, args...), "module", "simulator")
}

func normalizeRequest(request Request) Request {
	if request.ChargerCount == 0 {
		request.ChargerCount = defaultChargerCount
	}
	if request.ConnectorCount == 0 {
		request.ConnectorCount = defaultConnectorCount
	}
	if request.LoadCurve == "" {
		request.LoadCurve = defaultLoadCurve
	}
	return request
}

func defaultRequest() Request {
	return Request{
		ChargerCount:   defaultChargerCount,
		ConnectorCount: defaultConnectorCount,
		OnlineRate:     defaultOnlineRate,
		FaultRate:      defaultFaultRate,
		LoadCurve:      defaultLoadCurve,
	}
}
