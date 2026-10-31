package main

import (
	"runtime/debug"
	"strings"

	"github.com/lostcoords/wariwari-cli/internal/commands"
)

var version = "dev"

func main() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		}
	}

	commands.SetVersion(version)
	commands.Execute()
}
