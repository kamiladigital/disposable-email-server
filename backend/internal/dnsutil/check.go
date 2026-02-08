package dnsutil

import (
	"fmt"
	"net"
	"strings"
)

func CheckARecord(fqdn, expectedIP string) (bool, string) {
	ips, err := net.LookupHost(fqdn)
	if err != nil {
		return false, fmt.Sprintf("✗ FAILED (not found)")
	}
	for _, ip := range ips {
		if ip == expectedIP {
			return true, "✓ OK"
		}
	}
	return false, fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(ips, ", "), expectedIP)
}

func CheckMXRecord(fqdn, expectedFQDN string) (bool, string) {
	mxRecords, err := net.LookupMX(fqdn)
	if err != nil {
		return false, "✗ FAILED (not found)"
	}
	for _, mx := range mxRecords {
		mxHost := strings.TrimSuffix(mx.Host, ".")
		if mxHost == expectedFQDN {
			return true, "✓ OK"
		}
	}
	var hosts []string
	for _, mx := range mxRecords {
		hosts = append(hosts, strings.TrimSuffix(mx.Host, "."))
	}
	return false, fmt.Sprintf("✗ FAILED (points to %s, expected %s)", strings.Join(hosts, ", "), expectedFQDN)
}

// CheckMXRecordWithIP checks if any MX record for the domain resolves to the expected IP
// This is the proper way to validate email domains: domain → MX → mail server → IP
func CheckMXRecordWithIP(domain, expectedIP string) (bool, string, []string) {
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return false, "✗ FAILED (no MX records found)", nil
	}

	if len(mxRecords) == 0 {
		return false, "✗ FAILED (no MX records configured)", nil
	}

	var mxHosts []string
	var validMXHosts []string

	for _, mx := range mxRecords {
		mxHost := strings.TrimSuffix(mx.Host, ".")
		mxHosts = append(mxHosts, mxHost)

		// Check if this MX host resolves to the expected IP
		ips, err := net.LookupHost(mxHost)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			if ip == expectedIP {
				validMXHosts = append(validMXHosts, mxHost)
				break
			}
		}
	}

	if len(validMXHosts) > 0 {
		return true, fmt.Sprintf("✓ OK (MX: %s → %s)", strings.Join(validMXHosts, ", "), expectedIP), mxHosts
	}

	return false, fmt.Sprintf("✗ FAILED (MX records %s do not resolve to %s)", strings.Join(mxHosts, ", "), expectedIP), mxHosts
}
