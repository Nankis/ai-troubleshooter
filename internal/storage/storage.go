package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/ginseng/ai-troubleshooter/internal/caseflow"
	"github.com/ginseng/ai-troubleshooter/internal/config"
	mysqlstore "github.com/ginseng/ai-troubleshooter/internal/storage/mysql"
)

type OpenedStore struct {
	Store caseflow.Store
	Close func() error
}

func Open(ctx context.Context, cfg config.DatabaseConfig) (OpenedStore, error) {
	if strings.EqualFold(cfg.Driver, "mysql") && strings.TrimSpace(cfg.DSN) != "" {
		store, err := mysqlstore.New(ctx, cfg.DSN)
		if err != nil {
			return OpenedStore{}, err
		}
		return OpenedStore{Store: store, Close: store.Close}, nil
	}
	if cfg.DSN != "" && !strings.EqualFold(cfg.Driver, "memory") {
		return OpenedStore{}, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
	return OpenedStore{Store: caseflow.NewInMemoryStore(), Close: func() error { return nil }}, nil
}
