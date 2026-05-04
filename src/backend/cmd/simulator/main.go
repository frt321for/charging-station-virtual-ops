package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charging-ops/backend/internal/simulator"
)

func main() {
	cfg := simulator.Config{}
	flag.StringVar(&cfg.Mode, "mode", "once", "simulation mode: once or run")
	flag.StringVar(&cfg.APIBase, "api-base", "http://127.0.0.1:8080", "backend API base URL")
	flag.StringVar(&cfg.GroupCode, "group-code", "N-A1", "charger group code")
	flag.StringVar(&cfg.Prefix, "prefix", "SIM", "generated charger code prefix")
	flag.IntVar(&cfg.ChargerCount, "chargers", 1, "number of virtual chargers")
	flag.IntVar(&cfg.ConnectorCount, "connectors", 1, "connectors per charger")
	flag.Float64Var(&cfg.OnlineRate, "online-rate", 1, "online probability from 0 to 1")
	flag.Float64Var(&cfg.FaultRate, "fault-rate", 0, "fault probability from 0 to 1")
	flag.StringVar(&cfg.LoadCurve, "load-curve", "flat", "load curve: flat, commute, or random")
	flag.IntVar(&cfg.Ticks, "ticks", 5, "meter samples per once scenario")
	flag.DurationVar(&cfg.Interval, "interval", 5*time.Second, "heartbeat or meter sample interval")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := simulator.NewClient(cfg.APIBase, 15*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	runner := simulator.NewRunner(cfg, client, func(format string, args ...any) {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	})

	switch cfg.Mode {
	case simulator.ModeOnce:
		results, err := runner.RunOnce(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case simulator.ModeRun:
		if err := runner.RunContinuous(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q\n", cfg.Mode)
		os.Exit(2)
	}
}
