// Package main starts the mch terminal application.
package main

import (
	"cli/internal/app"
	"fmt"
	"os"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
