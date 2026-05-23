package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Nankis/ai-troubleshooter/internal/audit"
	"github.com/Nankis/ai-troubleshooter/internal/capability"
	"github.com/Nankis/ai-troubleshooter/internal/caseflow"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	mysqlstore "github.com/Nankis/ai-troubleshooter/internal/storage/mysql"
)

type OpenedStore struct {
	Store           caseflow.Store
	AuditSink       audit.Sink
	CapabilityStore capability.Store
	Close           func() error
}

func Open(ctx context.Context, cfg config.DatabaseConfig) (OpenedStore, error) {
	driver := strings.TrimSpace(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)

	if strings.EqualFold(driver, "memory") {
		if dsn != "" {
			return OpenedStore{}, fmt.Errorf("memory database driver does not accept DB_DSN")
		}
		return OpenedStore{
			Store:           caseflow.NewInMemoryStore(),
			AuditSink:       audit.NewMemorySink(),
			CapabilityStore: capability.NewMemoryStore(),
			Close:           func() error { return nil },
		}, nil
	}

	if strings.EqualFold(driver, "mysql") {
		if dsn == "" {
			return OpenedStore{}, fmt.Errorf("DB_DSN is required when DB_DRIVER=mysql; use DB_DRIVER=memory only for explicit local smoke tests")
		}
		store, err := mysqlstore.New(ctx, dsn)
		if err != nil {
			return OpenedStore{}, err
		}
		return OpenedStore{Store: store, AuditSink: store, CapabilityStore: store, Close: store.Close}, nil
	}

	return OpenedStore{}, fmt.Errorf("unsupported database driver %q", cfg.Driver)
}
