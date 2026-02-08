package dnsutil

import (
	"fmt"
	"net"
	"strings"
)

func ValidateFQDN(fqdn string) error {
	if fqdn == "" {
		return fmt.Errorf("FQDN cannot be empty")
	}
	if len(fqdn) > 253 {
		return fmt.Errorf("FQDN too long (max 253 characters)")
	}
	parts := strings.Split(fqdn, ".")
	if len(parts) < 2 {
		return fmt.Errorf("FQDN must have at least 2 parts (e.g., mail.example.com)")
	}
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("FQDN has empty labels")
		}
		if len(part) > 63 {
			return fmt.Errorf("FQDN label too long: %s (max 63 characters)", part)
		}
		if strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
			return fmt.Errorf("FQDN labels cannot start or end with hyphen: %s", part)
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return fmt.Errorf("FQDN contains invalid character: %c", r)
			}
		}
	}
	return nil
}

func ValidateIPv4(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IPv4 address")
	}
	return nil
}
