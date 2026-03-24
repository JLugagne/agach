package main

import (
	"fmt"
	"os"

	daemon "github.com/JLugagne/agach-mcp/internal/daemon"
)

func main() {
	if err := daemon.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}
}
