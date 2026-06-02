package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/NoorDigitalAgency/vakt/internal/app"
	"github.com/NoorDigitalAgency/vakt/internal/buildinfo"
	"github.com/NoorDigitalAgency/vakt/internal/config"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	command := "run"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}

	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stdout)
	configPath := flags.String("config", "/etc/vakt/config.yaml", "Path to the YAML config file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	switch command {
	case "help", "--help", "-h":
		usage(flags)
		return nil
	case "version":
		_, err := fmt.Fprintln(stdout, buildinfo.Version)
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	switch command {
	case "run":
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		service := app.NewService(cfg)
		return service.Run(ctx)
	case "snapshot":
		service := app.NewService(cfg)
		return service.SendManualSnapshot(context.Background())
	case "validate-config":
		_, err := fmt.Fprintf(stdout, "config %s is valid\n", *configPath)
		return err
	default:
		usage(flags)
		return errors.New("unknown command")
	}
}

func usage(flags *flag.FlagSet) {
	fmt.Fprintf(flags.Output(), "vakt monitors Linux server resources and posts snapshots to Slack.\n\n")
	fmt.Fprintf(flags.Output(), "Usage:\n")
	fmt.Fprintf(flags.Output(), "  vakt [run] --config /etc/vakt/config.yaml\n")
	fmt.Fprintf(flags.Output(), "  vakt snapshot --config /etc/vakt/config.yaml\n")
	fmt.Fprintf(flags.Output(), "  vakt validate-config --config /etc/vakt/config.yaml\n")
	fmt.Fprintf(flags.Output(), "  vakt version\n\n")
	flags.PrintDefaults()
}
