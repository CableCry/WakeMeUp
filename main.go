package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	tea "charm.land/bubbletea/v2"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}

	switch args[0] {
	case "list", "ls":
		os.Exit(cmdList())
	case "wake", "w":
		os.Exit(cmdWake(args[1:]))
	case "version", "-v", "--version":
		fmt.Println("wake " + version)
	case "help", "-h", "--help":
		printUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "wake: unknown command %q\n\n", args[0])
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

func runTUI() {
	devices, err := LoadDevices()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wake: could not load devices:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(newModel(devices)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "wake:", err)
		os.Exit(1)
	}
}

func cmdList() int {
	devices, err := LoadDevices()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wake: could not load devices:", err)
		return 1
	}
	if len(devices) == 0 {
		fmt.Println("No devices saved yet. Run `wake` to add one.")
		return 0
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tMAC\tTARGET")
	for _, d := range devices {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Name, d.MAC, d.target())
	}
	tw.Flush()
	return 0
}

func cmdWake(names []string) int {
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "usage: wake wake <name>... | all")
		return 2
	}

	devices, err := LoadDevices()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wake: could not load devices:", err)
		return 1
	}

	var targets []Device
	if len(names) == 1 && names[0] == "all" {
		targets = devices
	} else {
		for _, name := range names {
			idx := findDevice(devices, name)
			if idx < 0 {
				fmt.Fprintf(os.Stderr, "wake: no device named %q\n", name)
				return 1
			}
			targets = append(targets, devices[idx])
		}
	}

	if len(targets) == 0 {
		fmt.Println("No devices to wake.")
		return 0
	}

	failures := 0
	for _, d := range targets {
		if err := SendMagicPacket(d); err != nil {
			fmt.Fprintf(os.Stderr, "✕ %s: %v\n", d.Name, err)
			failures++
			continue
		}
		fmt.Printf("✓ magic packet sent to %s (%s)\n", d.Name, d.target())
	}
	if failures > 0 {
		return 1
	}
	return 0
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `wake — Wake-on-LAN manager

Usage:
  wake                 Launch the interactive manager (add, edit, remove, wake)
  wake list            List saved devices
  wake wake <name>...  Wake one or more saved devices by name
  wake wake all        Wake every saved device
  wake version         Print the version
  wake help            Show this help
`)
}
