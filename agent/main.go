package main

import (
	"fmt"
	"os"

	"watchflare-agent/cmd"
	"watchflare-agent/logger"
)

func main() {
	logger.Init()

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
			cmd.AgentVersion = Version
			_ = cmd.Register() // Ignore return value when called directly
			return

		case "status":
			cmd.Status()
			return

		case "start":
			cmd.StartService()
			return

		case "stop":
			cmd.StopService()
			return

		case "restart":
			cmd.RestartService()
			return

		case "logs":
			cmd.Logs()
			return

		case "update":
			cmd.AgentVersion = Version
			cmd.Update()
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
	cmd.AgentVersion = Version
	cmd.Run()
}

func printHelp() {
	fmt.Println("Watchflare Agent - Server Monitoring Agent")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  watchflare-agent [command]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  Installation & Setup:")
	fmt.Println("    install       Install the agent as a system service")
	fmt.Println("    uninstall     Uninstall the agent")
	fmt.Println("    register      Register the agent with the backend")
	fmt.Println()
	fmt.Println("  Service Control:")
	fmt.Println("    status        Show agent status")
	fmt.Println("    start         Start the agent service")
	fmt.Println("    stop          Stop the agent service")
	fmt.Println("    restart       Restart the agent service")
	fmt.Println("    logs          Follow agent logs")
	fmt.Println()
	fmt.Println("  Updates:")
	fmt.Println("    update        Update the agent to the latest version")
	fmt.Println("    update --check  Check if an update is available without installing")
	fmt.Println()
	fmt.Println("  Other:")
	fmt.Println("    (no command)  Run agent in foreground (for testing)")
	fmt.Println("    help          Show this help message")
	fmt.Println("    version       Show version information")
	fmt.Println()
	fmt.Println("Installation:")
	fmt.Println("  sudo watchflare-agent install [options]")
	fmt.Println("    --token=TOKEN   Registration token")
	fmt.Println("    --host=HOST     Backend hostname (default: localhost)")
	fmt.Println("    --port=PORT     Backend port (default: 50051)")
	fmt.Println()
	fmt.Println("Service Management:")
	fmt.Println("  watchflare-agent status              # Check if agent is running")
	fmt.Println("  sudo watchflare-agent start          # Start the agent")
	fmt.Println("  sudo watchflare-agent stop           # Stop the agent")
	fmt.Println("  sudo watchflare-agent restart        # Restart the agent")
	fmt.Println("  watchflare-agent logs                # View logs (Ctrl+C to exit)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Install and register in one command")
	fmt.Println("  sudo watchflare-agent install --token=wf_reg_xxx --host=monitor.example.com")
	fmt.Println()
	fmt.Println("  # Check agent status")
	fmt.Println("  watchflare-agent status")
	fmt.Println()
	fmt.Println("  # Restart after configuration change")
	fmt.Println("  sudo watchflare-agent restart")
	fmt.Println()
}

// Version is set at build time via ldflags: -X 'main.Version=...'
var Version = "dev"

func printVersion() {
	fmt.Printf("Watchflare Agent v%s\n", Version)
	fmt.Println("https://watchflare.io")
}
