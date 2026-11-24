package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/carrier"
)

// runCarrier is the process behind a kernel-carrier tunnel; the panel starts it with the file it wrote.
func runCarrier(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: x-ui carrier <config.json>")
		os.Exit(2)
	}
	cfg, err := carrier.Load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "carrier:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := carrier.Run(ctx, carrier.IP{}, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "carrier:", err)
		os.Exit(1)
	}
}
