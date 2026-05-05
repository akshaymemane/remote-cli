package main

import (
	"fmt"
	"log"
	"os"

	"remote-cli/internal/agent"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "pair":
		relayURL := flagValue(os.Args[2:], "--relay", "-r")
		if relayURL == "" {
			cfg, _ := agent.LoadConfig()
			if cfg != nil {
				relayURL = cfg.RelayURL
			}
		}
		if relayURL == "" {
			fmt.Fprintln(os.Stderr, "usage: remote-cli pair --relay <relay-url>")
			os.Exit(1)
		}
		if err := agent.Pair(relayURL); err != nil {
			log.Fatal(err)
		}
		cfg, err := agent.LoadConfig()
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		fmt.Println("Starting agent… (Ctrl+C to stop — use 'remote-cli service install' for background autostart)")
		if err := agent.Run(cfg); err != nil {
			log.Fatal(err)
		}

	case "run":
		cfg, err := agent.LoadConfig()
		if err != nil {
			log.Fatalf("load config: %v", err)
		}
		if err := agent.Run(cfg); err != nil {
			log.Fatal(err)
		}

	case "status":
		cfg, err := agent.LoadConfig()
		if err != nil || cfg.DeviceToken == "" {
			fmt.Println("not paired")
			os.Exit(1)
		}
		fmt.Printf("paired as %q\ndevice id: %s\nrelay:     %s\n", cfg.DeviceName, cfg.DeviceID, cfg.RelayURL)

	case "unpair":
		path := agent.ConfigPath()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Fatalf("remove config: %v", err)
		}
		fmt.Println("unpaired — config removed")

	case "service":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: remote-cli service <install|uninstall|start|stop|logs>")
			os.Exit(1)
		}
		var err error
		switch os.Args[2] {
		case "install":
			err = agent.ServiceInstall()
		case "uninstall":
			err = agent.ServiceUninstall()
		case "start":
			err = agent.ServiceStart()
		case "stop":
			err = agent.ServiceStop()
		case "logs":
			err = agent.ServiceLogs()
		default:
			fmt.Fprintln(os.Stderr, "usage: remote-cli service <install|uninstall|start|stop|logs>")
			os.Exit(1)
		}
		if err != nil {
			log.Fatal(err)
		}

	case "mcp-server":
		socketPath := flagValue(os.Args[2:], "--socket")
		if socketPath == "" {
			fmt.Fprintln(os.Stderr, "usage: remote-cli mcp-server --socket <path>")
			os.Exit(1)
		}
		if err := agent.RunMCPPermissionServer(socketPath); err != nil {
			log.Fatal(err)
		}

	case "version", "--version", "-v":
		fmt.Println(version)

	default:
		usage()
		os.Exit(1)
	}
}

func flagValue(args []string, flags ...string) string {
	for i, arg := range args {
		for _, f := range flags {
			if arg == f && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func usage() {
	fmt.Fprintf(os.Stderr, `remote-cli %s

Commands:
  pair   --relay <url>         Pair this machine and start the agent (shows QR code)
  run                          Start the agent and connect to the relay
  status                       Show pairing status and config path
  unpair                       Remove pairing config from this machine
  service install              Install and start as a background service
  service uninstall            Stop and remove the background service
  service start|stop           Start or stop the background service
  service logs                 Stream service logs
  version                      Print version
`, version)
}
