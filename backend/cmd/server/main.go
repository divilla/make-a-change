// Command server runs the backend HTTP API.
package main

import (
	"context"
	"flag"
	"mch_api/pkg/config"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.New()

	portFlag := flag.String("port", "", "server port")
	dbFlag := flag.String("db", "", "database connection string")
	flag.Parse()
	if *portFlag != "" {
		cfg.Port = *portFlag
	}
	if *dbFlag != "" {
		cfg.ConnectionString = *dbFlag
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, func() (application, error) { return start(ctx, cfg) }); err != nil {
		log.Error().Err(err).Msg("server stopped")
		os.Exit(1)
	}
}
