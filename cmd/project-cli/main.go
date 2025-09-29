package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	root := "."

	var err error
	switch cmd {
	case "init":
		err = runInit(root)
	case "new-feature":
		err = runNewFeature(root)
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "project-cli: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		exitWithError(err)
	}
}

func printUsage() {
	fmt.Println("Usage: project-cli <command>")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  init          Initialise the project with a new Go module path")
	fmt.Println("  new-feature   Scaffold a new feature interactively")
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "project-cli: %v\n", err)
	os.Exit(1)
}
