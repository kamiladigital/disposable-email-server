package dnsutil

import (
	"fmt"
	"strings"
)

func PrintDNSRecords(fqdn, publicIP string) error {
	// Validate FQDN and IP (should already be done in main)
	// Check DNS records
	aOk, aStatus := CheckARecord(fqdn, publicIP)
	mxOk, mxStatus := CheckMXRecord(fqdn, fqdn)

	if !aOk || !mxOk {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("DNS RECORDS VERIFICATION FAILED")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println()
		fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
		fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
		fmt.Println(strings.Repeat("=", 80))
		return fmt.Errorf("DNS records not properly configured")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DNS RECORDS VERIFICATION")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println("TYPE   | NAME                      | VALUE                                      | STATUS")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("A      | %-25s | %-40s | %s\n", fqdn, publicIP, aStatus)
	fmt.Printf("MX     | %-25s | %-40s | %s\n", fqdn, fqdn, mxStatus)
	fmt.Println(strings.Repeat("=", 80) + "\n")
	return nil
}
