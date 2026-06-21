package scanner_test

import (
	"context"
	"testing"

	"asset-api/internal/scanner"
)

func TestDNSScanner_Scan(t *testing.T) {
	s := scanner.NewDNSScanner()
	ctx := context.Background()

	res, err := s.Scan(ctx, "localhost")
	if err != nil {
		t.Fatalf("DNS scan failed: %v", err)
	}

	dnsRes, ok := res.(scanner.DNSResult)
	if !ok {
		t.Fatalf("Expected DNSResult, got %T", res)
	}

	if dnsRes.Domain != "localhost" {
		t.Errorf("Expected domain 'localhost', got '%s'", dnsRes.Domain)
	}
}

func TestPortScanner_SafetyCheck(t *testing.T) {
	s := scanner.NewPortScanner()
	ctx := context.Background()

	// Test public IP (8.8.8.8) - should be rejected!
	_, err := s.Scan(ctx, "8.8.8.8")
	if err == nil {
		t.Error("Expected error when scanning public IP 8.8.8.8, but scan succeeded")
	}

	// Test private IP (127.0.0.1) - should be allowed!
	res, err := s.Scan(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Localhost port scan failed: %v", err)
	}

	portRes, ok := res.(scanner.PortScanResult)
	if !ok {
		t.Fatalf("Expected PortScanResult, got %T", res)
	}

	if portRes.IPAddress != "127.0.0.1" {
		t.Errorf("Expected IP '127.0.0.1', got '%s'", portRes.IPAddress)
	}
}

func TestSSLScanner_Fallback(t *testing.T) {
	s := scanner.NewSSLScanner()
	ctx := context.Background()

	// Dial to localhost which usually won't have SSL/TLS on 443, should trigger fallback
	res, err := s.Scan(ctx, "localhost")
	if err != nil {
		t.Fatalf("SSL scan error: %v", err)
	}

	sslRes, ok := res.(scanner.SSLScanResult)
	if !ok {
		t.Fatalf("Expected SSLScanResult, got %T", res)
	}

	if sslRes.Domain != "localhost" {
		t.Errorf("Expected domain 'localhost', got '%s'", sslRes.Domain)
	}
	if sslRes.Certificate.Issuer == "" {
		t.Error("Expected Certificate Issuer info")
	}
}

func TestTechScanner_Fallback(t *testing.T) {
	s := scanner.NewTechScanner()
	ctx := context.Background()

	res, err := s.Scan(ctx, "localhost")
	if err != nil {
		t.Fatalf("Tech scan error: %v", err)
	}

	techRes, ok := res.(scanner.TechScanResult)
	if !ok {
		t.Fatalf("Expected TechScanResult, got %T", res)
	}

	if techRes.Domain != "localhost" {
		t.Errorf("Expected domain 'localhost', got '%s'", techRes.Domain)
	}
	if len(techRes.Technologies) == 0 {
		t.Error("Expected at least one detected technology")
	}
}
