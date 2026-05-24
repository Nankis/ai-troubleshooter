package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/Nankis/ai-troubleshooter/internal/config"
)

func TestOpenRequiresDSNForMySQL(t *testing.T) {
	_, err := Open(context.Background(), config.DatabaseConfig{Driver: "mysql"})
	if err == nil || !strings.Contains(err.Error(), "DB_DSN is required") {
		t.Fatalf("expected DB_DSN required error, got %v", err)
	}
}

func TestOpenRequiresExplicitMemoryDriver(t *testing.T) {
	opened, err := Open(context.Background(), config.DatabaseConfig{Driver: "memory"})
	if err != nil {
		t.Fatalf("open memory store: %v", err)
	}
	if opened.AuditSink == nil || opened.CapabilityStore == nil || opened.Close == nil {
		t.Fatalf("expected memory audit sink, capability store, and close function")
	}
}

func TestOpenRejectsMemoryDriverWithDSN(t *testing.T) {
	_, err := Open(context.Background(), config.DatabaseConfig{
		Driver: "memory",
		DSN:    "root:secret@tcp(127.0.0.1:3306)/ai_troubleshooter",
	})
	if err == nil || !strings.Contains(err.Error(), "does not accept DB_DSN") {
		t.Fatalf("expected memory dsn error, got %v", err)
	}
}

func TestValidateLocalMySQLDSNRejectsNonCanonicalSchema(t *testing.T) {
	t.Setenv("ALLOW_NON_CANONICAL_LOCAL_DB", "")
	err := validateLocalMySQLDSN("root:secret@tcp(127.0.0.1:3306)/ai_troubleshooter_itest")
	if err == nil || !strings.Contains(err.Error(), "local MySQL platform database") {
		t.Fatalf("expected local schema guard error, got %v", err)
	}
}

func TestValidateLocalMySQLDSNRejectsNonCanonicalLocalhostSchema(t *testing.T) {
	t.Setenv("ALLOW_NON_CANONICAL_LOCAL_DB", "")
	err := validateLocalMySQLDSN("root:secret@tcp(localhost:3306)/ai_troubleshooter_itest")
	if err == nil || !strings.Contains(err.Error(), "local MySQL platform database") {
		t.Fatalf("expected localhost schema guard error, got %v", err)
	}
}

func TestValidateLocalMySQLDSNRejectsNonCanonicalIPv6Schema(t *testing.T) {
	t.Setenv("ALLOW_NON_CANONICAL_LOCAL_DB", "")
	err := validateLocalMySQLDSN("root:secret@tcp([::1]:3306)/ai_troubleshooter_itest")
	if err == nil || !strings.Contains(err.Error(), "local MySQL platform database") {
		t.Fatalf("expected ipv6 local schema guard error, got %v", err)
	}
}

func TestValidateLocalMySQLDSNAllowsCanonicalSchema(t *testing.T) {
	t.Setenv("ALLOW_NON_CANONICAL_LOCAL_DB", "")
	err := validateLocalMySQLDSN("root:secret@tcp(127.0.0.1:3306)/ai_troubleshooter")
	if err != nil {
		t.Fatalf("canonical local schema should be allowed: %v", err)
	}
}

func TestValidateLocalMySQLDSNAllowsExplicitNonCanonicalSchema(t *testing.T) {
	t.Setenv("ALLOW_NON_CANONICAL_LOCAL_DB", "true")
	err := validateLocalMySQLDSN("root:secret@tcp(127.0.0.1:3306)/ai_troubleshooter_itest")
	if err != nil {
		t.Fatalf("explicit non-canonical local schema should be allowed: %v", err)
	}
}

func TestOpenRejectsUnsupportedDriver(t *testing.T) {
	_, err := Open(context.Background(), config.DatabaseConfig{Driver: "sqlite"})
	if err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("expected unsupported driver error, got %v", err)
	}
}
