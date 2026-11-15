package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/enrolment"
)

// joinMainPanel runs "x-ui enrol <link>" on a node; Xray must be restarted afterwards to open the tunnel.
func joinMainPanel(args []string) {
	if len(args) != 1 {
		fmt.Println("usage: x-ui enrol <dalan://enrol/... link from the main panel>")
		os.Exit(2)
	}
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("database initialization failed:", err)
		os.Exit(1)
	}
	if err := (&enrolment.Service{}).Join(context.Background(), args[0]); err != nil {
		fmt.Println("enrolment failed:", err)
		os.Exit(1)
	}
	fmt.Println("This server joined the main panel. It appears there once Xray has restarted here.")
}
