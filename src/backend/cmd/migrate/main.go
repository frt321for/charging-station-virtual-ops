package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"charging-ops/backend/internal/common/config"
	"charging-ops/backend/internal/platform/database"
	"charging-ops/backend/internal/platform/migrations"
)

func main() {
	action := flag.String("action", "status", "Migration action: status, up, or down")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := config.Load()
	databaseClient, err := database.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("failed to connect database", "module", "migration", "traceId", "migration", "error", err)
		os.Exit(1)
	}
	defer databaseClient.Close()

	runner := migrations.NewRunner(databaseClient)
	if err := run(ctx, runner, *action); err != nil {
		logger.Error("migration failed", "module", "migration", "traceId", "migration", "action", *action, "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, runner *migrations.Runner, action string) error {
	switch action {
	case "status":
		statuses, err := runner.Status(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied {
				state = "applied"
			}
			fmt.Printf("%s %-24s %s\n", status.Version, status.Name, state)
		}
	case "up":
		changed, err := runner.Up(ctx)
		if err != nil {
			return err
		}
		printChanges("applied", changed)
	case "down":
		changed, err := runner.Down(ctx)
		if err != nil {
			return err
		}
		printChanges("rolled back", changed)
	default:
		return fmt.Errorf("unsupported migration action %q", action)
	}

	return nil
}

func printChanges(prefix string, changed []string) {
	if len(changed) == 0 {
		fmt.Println("no migrations changed")
		return
	}
	for _, migration := range changed {
		fmt.Printf("%s %s\n", prefix, migration)
	}
}
