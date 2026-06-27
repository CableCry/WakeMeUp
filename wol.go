package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const defaultPort = 9

const broadcastAddr = "255.255.255.255"

func normalizeMAC(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("MAC address is required")
	}
	hw, err := net.ParseMAC(s)
	if err != nil {
		return "", fmt.Errorf("not a valid MAC address (expected e.g. AA:BB:CC:DD:EE:FF)")
	}
	if len(hw) != 6 {
		return "", fmt.Errorf("must be a 6-byte MAC address")
	}
	return hw.String(), nil
}

func buildMagicPacket(hw net.HardwareAddr) []byte {
	packet := make([]byte, 6+16*6)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], hw)
	}
	return packet
}

func SendMagicPacket(d Device) error {
	hw, err := net.ParseMAC(strings.TrimSpace(d.MAC))
	if err != nil {
		return fmt.Errorf("invalid MAC %q: %w", d.MAC, err)
	}
	if len(hw) != 6 {
		return fmt.Errorf("invalid MAC %q: must be 6 bytes", d.MAC)
	}

	addr, err := net.ResolveUDPAddr("udp", d.target())
	if err != nil {
		return fmt.Errorf("resolve %s: %w", d.target(), err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("open socket: %w", err)
	}
	defer conn.Close()

	packet := buildMagicPacket(hw)
	n, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("send packet: %w", err)
	}
	if n != len(packet) {
		return fmt.Errorf("short write: sent %d of %d bytes", n, len(packet))
	}
	return nil
}

func (d Device) target() string {
	host := strings.TrimSpace(d.IP)
	if host == "" {
		host = broadcastAddr
	}
	port := d.Port
	if port == 0 {
		port = defaultPort
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}
