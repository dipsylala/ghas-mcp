package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/dipsylala/ghas-mcp/internal/cli"
	"github.com/dipsylala/ghas-mcp/internal/server"
	tools "github.com/dipsylala/ghas-mcp/internal/tool_registry"
)

//go:embed tools.json
var toolsJSONData []byte

//go:embed instructions.json
var instructionsJSONData []byte

func init() {
	tools.SetToolsJSON(toolsJSONData)
	server.SetInstructions(instructionsJSONData)
}

// version is set at build time via -ldflags="-X main.version=x.y.z"
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	verbose := flag.Bool("verbose", false, "Enable verbose logging to stderr")
	logFile := flag.String("log", "", "Write logs to this file instead of stderr")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stderr, "ghas-mcp version %s\n", version)
		os.Exit(0)
	}

	if err := cli.ConfigureLogging(*logFile, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "logging setup error: %v\n", err)
		os.Exit(1)
	}

	s, err := server.NewMCPServer(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server init error: %v\n", err)
		os.Exit(1)
	}

	if err := s.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
