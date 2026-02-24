package cmd

import (
	"fmt"
	"os"

	"canvas-cli/internal/ui"
)

func runDebugLogin() {
	fmt.Println()
	ui.Header("Debug Connection Test")
	fmt.Println()

	client.Debug = true
	name, err := client.TestConnection()
	if err != nil {
		fmt.Println()
		ui.Error(err.Error())
		os.Exit(1)
	}

	fmt.Println()
	if name != "" {
		ui.Success(fmt.Sprintf("Connection successful! Authenticated as: %s", name))
	} else {
		ui.Success("Connection successful! API access confirmed.")
	}
	fmt.Println()
}
