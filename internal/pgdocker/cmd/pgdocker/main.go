// pgdocker starts a Postgres Docker container and exposes it to a wrapped
// command through the PGURL environment variable.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/meoyawn/pggen/internal/errs"
	"github.com/meoyawn/pggen/internal/pgdocker"
)

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal(err)
	}
}

func run() (mErr error) {
	if len(os.Args) < 3 || os.Args[1] != "--" {
		return fmt.Errorf("usage: %s -- <command> [args...]", os.Args[0])
	}

	ctx := context.Background()
	docker, err := pgdocker.Start(ctx, nil)
	if err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}
	defer errs.Capture(&mErr, func() error {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return docker.Stop(stopCtx)
	}, "stop postgres")

	connStr, err := docker.ConnString()
	if err != nil {
		return fmt.Errorf("postgres connection string: %w", err)
	}

	wrapped := exec.Command(os.Args[2], os.Args[3:]...) //nolint:gosec // Runs the command explicitly supplied after --.
	wrapped.Stdin = os.Stdin
	wrapped.Stdout = os.Stdout
	wrapped.Stderr = os.Stderr
	wrapped.Env = append(os.Environ(), "PGURL="+connStr)
	if err := wrapped.Run(); err != nil {
		return fmt.Errorf("%v: %w", wrapped.Args, err)
	}
	return nil
}
