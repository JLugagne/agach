// Command qa-seed wipes and re-seeds the database with deterministic test data
// for Playwright E2E tests. It exits 0 on success and prints the IDs of every
// seeded entity as JSON so test harnesses can consume them.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/qaseed"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to database")
	}
	defer pool.Close()

	result, err := qaseed.Run(ctx, pool, logger)
	if err != nil {
		logger.WithError(err).Fatal("seeding failed")
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.WithError(err).Fatal("failed to marshal result")
	}
	fmt.Println(string(out))
	os.Exit(0)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
