package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"
)

const serviceName = "clauductor"

const serviceTemplate = `[Unit]
Description=Clauductor - Claude Web UI
After=network.target

[Service]
Type=simple
ExecStart={{.ExecPath}} --host={{.Host}} --port={{.Port}}
Restart=on-failure
RestartSec=5
Environment=HOME={{.Home}}

[Install]
WantedBy=default.target
`

func serviceFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", serviceName+".service")
}

func runServiceCommand(args []string) {
	if runtime.GOOS != "linux" {
		fmt.Println("Service management is only supported on Linux (systemd).")
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Println("Usage: clauductor service <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  install [--host=0.0.0.0] [--port=3003]  Install systemd user service")
		fmt.Println("  start                                     Start the service")
		fmt.Println("  stop                                      Stop the service")
		fmt.Println("  restart                                   Restart the service")
		fmt.Println("  enable                                    Enable autostart on login")
		fmt.Println("  disable                                   Disable autostart")
		fmt.Println("  status                                    Show service status")
		os.Exit(1)
	}

	switch args[0] {
	case "install":
		serviceInstall(args[1:])
	case "start":
		serviceCtl("start", true)
	case "stop":
		serviceCtl("stop", true)
	case "restart":
		serviceCtl("restart", true)
	case "enable":
		serviceCtl("enable", true)
	case "disable":
		serviceCtl("disable", true)
	case "status":
		serviceCtl("status", false)
	default:
		fmt.Printf("Unknown service command: %s\n", args[0])
		fmt.Println("Run 'clauductor service' to see available commands.")
		os.Exit(1)
	}
}

func serviceInstall(args []string) {
	fs := flag.NewFlagSet("service install", flag.ExitOnError)
	host := fs.String("host", "0.0.0.0", "Host to bind to")
	port := fs.String("port", "3003", "Port to listen on")
	fs.Parse(args)

	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	execPath, _ = filepath.Abs(execPath)

	home, _ := os.UserHomeDir()

	serviceDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		fmt.Printf("Error creating systemd user dir: %v\n", err)
		os.Exit(1)
	}

	tmpl := template.Must(template.New("service").Parse(serviceTemplate))
	f, err := os.Create(serviceFilePath())
	if err != nil {
		fmt.Printf("Error creating service file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	tmpl.Execute(f, map[string]string{
		"ExecPath": execPath,
		"Host":     *host,
		"Port":     *port,
		"Home":     home,
	})

	fmt.Printf("Service file written to: %s\n", serviceFilePath())
	fmt.Printf("Binding to:              %s:%s\n", *host, *port)
	fmt.Println()

	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		fmt.Printf("Warning: daemon-reload failed: %v\n%s\n", err, out)
	} else {
		fmt.Println("systemd daemon reloaded.")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  clauductor service enable   # autostart on login")
	fmt.Println("  clauductor service start    # start now")
}

func serviceCtl(cmd string, silent bool) {
	c := exec.Command("systemctl", "--user", cmd, serviceName)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil && cmd != "status" {
		fmt.Printf("Error running 'systemctl --user %s %s': %v\n", cmd, serviceName, err)
		os.Exit(1)
	}
	if silent && cmd != "status" {
		fmt.Printf("Service %s: %s\n", serviceName, cmd)
	}
}
