// Command tamagotchi-go runs the Tamagotchi CLI game.
package main

import (
	"os"

	"github.com/leekli/tamagotchi-go/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
