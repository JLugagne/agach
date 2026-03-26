package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JLugagne/agach-mcp/internal/sidecar"
)

func main() {
	mode := flag.String("mode", "", "Tool mode: 'pm' for planning-only tools, empty for all tools")
	flag.Parse()

	if err := sidecar.Run(*mode); err != nil {
		fmt.Fprintf(os.Stderr, "agach-sidecar: %v\n", err)
		os.Exit(1)
	}
}
