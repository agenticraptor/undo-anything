// Command ua is the undo-anything CLI: a universal local time machine for any
// folder. See https://github.com/agenticraptor/undo-anything for documentation.
package main

import (
	"os"

	"github.com/agenticraptor/undo-anything/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
