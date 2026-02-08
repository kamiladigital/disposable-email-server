package dnsutil

import "testing"

func TestValidateFQDN(t *testing.T) {
	cases := []struct {
		fqdn string
		ok   bool
	}{
		{"mail.example.com", true},
		{"a.com", true},
		{"", false},
		{"a", false},
		{"-bad.com", false},
		{"bad-.com", false},
		{"bad..com", false},
	}
	for _, c := range cases {
		err := ValidateFQDN(c.fqdn)
		if (err == nil) != c.ok {
			t.Errorf("ValidateFQDN(%q) = %v, want ok=%v", c.fqdn, err, c.ok)
		}
	}
}

func TestValidateIPv4(t *testing.T) {
	cases := []struct {
		ip string
		ok bool
	}{
		{"1.2.3.4", true},
		{"127.0.0.1", true},
		{"", false},
		{"999.1.1.1", false},
		{"abc.def.ghi.jkl", false},
	}
	for _, c := range cases {
		err := ValidateIPv4(c.ip)
		if (err == nil) != c.ok {
			t.Errorf("ValidateIPv4(%q) = %v, want ok=%v", c.ip, err, c.ok)
		}
	}
}
