// Package cli provides argument parsing and logging configuration for the ghas-mcp server.
package cli

import (
	"fmt"
	"io"
	"log"
	"os"
)

// ConfigureLogging sets up logging output based on flags.
// If logFilePath is non-empty, logs are written there.
// If verbose is false and no log file is specified, logs are discarded.
func ConfigureLogging(logFilePath string, verbose bool) error {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if logFilePath != "" {
		// #nosec G304 -- logFilePath comes from a CLI flag; user controls the destination.
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
		}
		log.SetOutput(f)
		return nil
	}

	if !verbose {
		log.SetOutput(io.Discard)
	}
	// verbose + no log file → logs go to stderr (Go default)
	return nil
}
