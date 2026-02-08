package dnsutil

import (
	"testing"
)

func TestCheckARecord_Localhost(t *testing.T) {
	ok, msg := CheckARecord("localhost", "127.0.0.1")
	if !ok {
		t.Errorf("CheckARecord failed for localhost: %s", msg)
	}
}

func TestCheckARecord_MismatchedIP(t *testing.T) {
	ok, msg := CheckARecord("localhost", "1.1.1.1")
	if ok {
		t.Errorf("CheckARecord should fail for mismatched IP: %s", msg)
	}
}

func TestCheckMXRecord_InvalidDomain(t *testing.T) {
	ok, msg := CheckMXRecord("invalid-domain-xyz-9999.test", "invalid-domain-xyz-9999.test")
	if ok {
		t.Errorf("CheckMXRecord should fail for invalid domain: %s", msg)
	}
	if msg == "" {
		t.Error("CheckMXRecord should return error message")
	}
}

func TestCheckMXRecordWithIP_InvalidDomain(t *testing.T) {
	ok, msg, mxHosts := CheckMXRecordWithIP("invalid-domain-xyz-9999.test", "1.2.3.4")
	if ok {
		t.Errorf("CheckMXRecordWithIP should fail for invalid domain: %s", msg)
	}
	if msg == "" {
		t.Error("CheckMXRecordWithIP should return error message")
	}
	if mxHosts != nil {
		t.Logf("MX hosts: %v", mxHosts)
	}
}
