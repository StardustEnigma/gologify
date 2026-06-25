package main

import (
	"os"

	"github.com/StardustEnigma/gologify/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
