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

func TestOpenRejectsUnsupportedDriver(t *testing.T) {
	_, err := Open(context.Background(), config.DatabaseConfig{Driver: "sqlite"})
	if err == nil || !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("expected unsupported driver error, got %v", err)
	}
}
