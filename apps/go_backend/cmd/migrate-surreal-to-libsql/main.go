// Package main implements a one-shot migration tool that reads a SurrealDB
// JSON export and writes the data into the libSQL schema produced by
// db/migrations/*.sql.
//
// It is an offline converter: SurrealDB is NOT queried live. Produce the
// export with `surreal sql --json` (see MIGRATING_FROM_SURREAL.md), then run
// this tool against a fresh libSQL database file.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "turso.tech/database/tursogo"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	var (
		inputPath  = flag.String("input", "", "Path to SurrealDB JSON export (bundle {\"tables\":{...}} or raw `surreal sql --json` output)")
		dbPath     = flag.String("db", "", "Target libSQL database file path (created if missing)")
		migrations = flag.String("migrations", "db/migrations", "Path to libSQL migrations directory")
		dryRun     = flag.Bool("dry-run", false, "Parse and validate only; do not write to the database")
		limit      = flag.Int("limit", 0, "Max rows to import per table (0 = no limit)")
		tablesFlag = flag.String("tables", "", "Comma-separated list of tables to import (default: all known tables)")
		verbose    = flag.Bool("v", false, "Verbose logging")
	)
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if *inputPath == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		flag.Usage()
		os.Exit(2)
	}
	if *dbPath == "" && !*dryRun {
		fmt.Fprintln(os.Stderr, "error: --db is required (unless --dry-run)")
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()

	// --- Parse the export -------------------------------------------------
	exp, err := ParseExport(*inputPath)
	if err != nil {
		log.Fatal().Err(err).Str("input", *inputPath).Msg("failed to parse export")
	}
	log.Info().Int("tables", len(exp.Tables)).Msg("export parsed")
	for name, rows := range exp.Tables {
		log.Info().Str("table", name).Int("rows", len(rows)).Msg("  source table")
	}

	// Table filter
	var only map[string]bool
	if *tablesFlag != "" {
		only = map[string]bool{}
		for _, t := range strings.Split(*tablesFlag, ",") {
			only[strings.TrimSpace(t)] = true
		}
	}

	// --- Open target DB ---------------------------------------------------
	var db *database.DB
	if !*dryRun {
		absDB, err := filepath.Abs(*dbPath)
		if err != nil {
			log.Fatal().Err(err).Msg("resolving db path")
		}
		absMig, err := filepath.Abs(*migrations)
		if err != nil {
			log.Fatal().Err(err).Msg("resolving migrations path")
		}
		db, err = database.New(ctx, database.Config{
			URL:            absDB,
			MigrationsPath: absMig,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open target database")
		}
		defer db.Close(ctx)
		log.Info().Str("db", absDB).Msg("target database opened (migrations applied)")
	} else {
		log.Info().Msg("dry-run mode: no database writes")
	}

	// --- Import -----------------------------------------------------------
	imp := &Importer{
		db:      db,
		dryRun:  *dryRun,
		limit:   *limit,
		only:    only,
		summary: &ImportSummary{Tables: map[string]*TableSummary{}},
	}
	if err := imp.Run(ctx, exp); err != nil {
		log.Error().Err(err).Msg("import finished with errors")
		imp.summary.Print()
		os.Exit(1)
	}
	imp.summary.Print()
}
