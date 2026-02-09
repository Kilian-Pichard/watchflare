package main

import (
	"fmt"
	"log"
	"os"

	"watchflare-agent/cmd"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	// Check for subcommands
	if len(os.Args) > 1 {
		subcommand := os.Args[1]

		switch subcommand {
		case "install":
			cmd.Install()
			return

		case "uninstall":
			cmd.Uninstall()
			return

		case "register":
			_ = cmd.Register() // Ignore return value when called directly
			return

		case "help", "-h", "--help":
			printHelp()
			return

		case "version", "-v", "--version":
			printVersion()
			return

		default:
			fmt.Printf("Unknown command: %s\n\n", subcommand)
			printHelp()
			os.Exit(1)
		}
	}

	// No subcommand = run normal agent
	cmd.Run()
}

func printHelp() {
	fmt.Println("Watchflare Agent - Server Monitoring Agent")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  watchflare-agent [command]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  (no command)  Start the agent (default)")
	fmt.Println("  install       Install the agent as a system service")
	fmt.Println("  uninstall     Uninstall the agent")
	fmt.Println("  register      Register the agent with the backend")
	fmt.Println("  help          Show this help message")
	fmt.Println("  version       Show version information")
	fmt.Println()
	fmt.Println("Installation:")
	fmt.Println("  sudo watchflare-agent install [options]")
	fmt.Println("    --token=TOKEN   Registration token")
	fmt.Println("    --host=HOST     Backend hostname (default: localhost)")
	fmt.Println("    --port=PORT     Backend port (default: 50051)")
	fmt.Println()
	fmt.Println("Registration:")
	fmt.Println("  sudo watchflare-agent register --token=TOKEN [options]")
	fmt.Println("    --token=TOKEN   Registration token (required)")
	fmt.Println("    --host=HOST     Backend hostname (default: localhost)")
	fmt.Println("    --port=PORT     Backend port (default: 50051)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Install and register in one command")
	fmt.Println("  sudo watchflare-agent install --token=wf_reg_xxx --host=monitor.example.com")
	fmt.Println()
	fmt.Println("  # Install only (register later)")
	fmt.Println("  sudo watchflare-agent install")
	fmt.Println()
	fmt.Println("  # Register separately")
	fmt.Println("  sudo watchflare-agent register --token=wf_reg_xxx --host=monitor.example.com")
	fmt.Println()
	fmt.Println("  # Run agent in foreground (for testing)")
	fmt.Println("  watchflare-agent")
	fmt.Println()
}

func printVersion() {
	fmt.Println("Watchflare Agent v1.0.0")
	fmt.Println("https://watchflare.io")
}
