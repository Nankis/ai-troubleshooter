package storage

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Nankis/ai-troubleshooter/internal/audit"
	"github.com/Nankis/ai-troubleshooter/internal/capability"
	"github.com/Nankis/ai-troubleshooter/internal/config"
	mysqlstore "github.com/Nankis/ai-troubleshooter/internal/storage/mysql"
)

type OpenedStore struct {
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
			AuditSink:       audit.NewMemorySink(),
			CapabilityStore: capability.NewMemoryStore(),
			Close:           func() error { return nil },
		}, nil
	}

	if strings.EqualFold(driver, "mysql") {
		if dsn == "" {
			return OpenedStore{}, fmt.Errorf("DB_DSN is required when DB_DRIVER=mysql; use DB_DRIVER=memory only for explicit local smoke tests")
		}
		if err := validateLocalMySQLDSN(dsn); err != nil {
			return OpenedStore{}, err
		}
		store, err := mysqlstore.New(ctx, dsn)
		if err != nil {
			return OpenedStore{}, err
		}
		return OpenedStore{AuditSink: store, CapabilityStore: store, Close: store.Close}, nil
	}

	return OpenedStore{}, fmt.Errorf("unsupported database driver %q", cfg.Driver)
}

var goMySQLDSNRE = regexp.MustCompile(`@tcp\(([^)]*)\)/([^?]+)`)

func validateLocalMySQLDSN(dsn string) error {
	host, database, ok := parseGoMySQLDSNHostDatabase(dsn)
	if !ok || !isLocalMySQLHost(host) || envTruthy("ALLOW_NON_CANONICAL_LOCAL_DB") {
		return nil
	}
	canonical := strings.TrimSpace(os.Getenv("MYSQL_CANONICAL_LOCAL_DATABASE"))
	if canonical == "" {
		canonical = "ai_troubleshooter"
	}
	if database != canonical {
		return fmt.Errorf(
			"local MySQL platform database must be %q; got %q; set ALLOW_NON_CANONICAL_LOCAL_DB=true only for intentional isolated experiments with a recorded cleanup plan",
			canonical,
			database,
		)
	}
	return nil
}

func parseGoMySQLDSNHostDatabase(dsn string) (string, string, bool) {
	match := goMySQLDSNRE.FindStringSubmatch(dsn)
	if len(match) != 3 {
		return "", "", false
	}
	hostPort := strings.TrimSpace(match[1])
	host := hostPort
	if strings.HasPrefix(hostPort, "[") {
		if end := strings.Index(hostPort, "]"); end > 0 {
			host = hostPort[1:end]
		}
	} else if strings.Count(hostPort, ":") <= 1 {
		if idx := strings.LastIndex(hostPort, ":"); idx > 0 {
			host = hostPort[:idx]
		}
	}
	return strings.TrimSpace(host), strings.TrimSpace(match[2]), true
}

func isLocalMySQLHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
