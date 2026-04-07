// Package scanner provides functionality to scan and detect open TCP/UDP ports
// on the local system by reading from /proc/net or using system calls.
package scanner

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Protocol represents a network protocol type.
type Protocol string

const (
	TCP  Protocol = "tcp"
	TCP6 Protocol = "tcp6"
	UDP  Protocol = "udp"
	UDP6 Protocol = "udp6"
)

// PortEntry represents a single open port detected on the system.
type PortEntry struct {
	Port     uint16
	Protocol Protocol
	Address  string
	PID      int
}

// String returns a human-readable representation of a PortEntry.
func (p PortEntry) String() string {
	return fmt.Sprintf("%s:%d (%s)", p.Address, p.Port, p.Protocol)
}

// Scanner scans the local system for open ports.
type Scanner struct {
	includeIPv6 bool
	includeUDP  bool
}

// New creates a new Scanner with the given options.
func New(includeIPv6, includeUDP bool) *Scanner {
	return &Scanner{
		includeIPv6: includeIPv6,
		includeUDP:  includeUDP,
	}
}

// Scan returns all currently open ports on the system.
func (s *Scanner) Scan() ([]PortEntry, error) {
	var entries []PortEntry

	procs := []struct {
		path  string
		proto Protocol
	}{
		{"/proc/net/tcp", TCP},
	}

	if s.includeIPv6 {
		procs = append(procs, struct {
			path  string
			proto Protocol
		}{"/proc/net/tcp6", TCP6})
	}
	if s.includeUDP {
		procs = append(procs, struct {
			path  string
			proto Protocol
		}{"/proc/net/udp", UDP})
	}
	if s.includeUDP && s.includeIPv6 {
		procs = append(procs, struct {
			path  string
			proto Protocol
		}{"/proc/net/udp6", UDP6})
	}

	for _, p := range procs {
		results, err := parseProcNet(p.path, p.proto)
		if err != nil {
			// Skip if the file doesn't exist (e.g., IPv6 not available)
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("scanning %s: %w", p.path, err)
		}
		entries = append(entries, results...)
	}

	return entries, nil
}

// parseProcNet reads a /proc/net/{tcp,udp} file and returns listening port entries.
// Only entries with state 0A (TCP_LISTEN) or 07 (UDP) are included.
func parseProcNet(path string, proto Protocol) ([]PortEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []PortEntry
	scanner := bufio.NewScanner(f)

	// Skip the header line
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		state := fields[3]
		// 0A = TCP LISTEN, 07 = UDP (stateless)
		if proto == TCP || proto == TCP6 {
			if state != "0A" {
				continue
			}
		}

		local := fields[1]
		parts := strings.Split(local, ":")
		if len(parts) != 2 {
			continue
		}

		portHex := parts[1]
		portNum, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil {
			continue
		}

		addrHex := parts[0]
		addr := hexToIP(addrHex)

		entries = append(entries, PortEntry{
			Port:     uint16(portNum),
			Protocol: proto,
			Address:  addr,
		})
	}

	return entries, scanner.Err()
}

// hexToIP converts a hex-encoded IP address from /proc/net format to a string.
func hexToIP(hex string) string {
	if len(hex) == 8 {
		// IPv4: little-endian 4-byte hex
		b := make([]byte, 4)
		for i := 0; i < 4; i++ {
			v, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
			if err != nil {
				return hex
			}
			b[3-i] = byte(v)
		}
		return net.IP(b).String()
	}
	// Return raw for IPv6 or unknown formats
	return hex
}
