package mysql

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationsFollowDatabaseTemplate(t *testing.T) {
	root := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	createTableRE := regexp.MustCompile("(?is)CREATE\\s+TABLE(?:\\s+IF\\s+NOT\\s+EXISTS)?\\s+`?([a-zA-Z0-9_]+)`?\\s*\\((.*?)\\)\\s*ENGINE=InnoDB\\s+DEFAULT\\s+CHARSET=utf8mb4\\s+COLLATE=utf8mb4_general_ci")
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sql := string(raw)
		if strings.Contains(sql, "created_at") || strings.Contains(sql, "updated_at") {
			t.Fatalf("%s uses created_at/updated_at; use create_time/update_time", entry.Name())
		}
		if strings.Contains(sql, "lark_message_id") || strings.Contains(sql, "lark_user_id") {
			t.Fatalf("%s uses lark-specific storage fields; use platform/caller fields", entry.Name())
		}
		if strings.Contains(sql, " JSON ") || strings.Contains(sql, "\tJSON ") {
			t.Fatalf("%s uses MySQL JSON type; use TEXT for *_json fields", entry.Name())
		}
		if regexp.MustCompile("`status`\\s+VARCHAR").MatchString(sql) {
			t.Fatalf("%s uses status as business status; use *_status and reserve status TINYINT for row status", entry.Name())
		}
		for _, match := range createTableRE.FindAllStringSubmatch(sql, -1) {
			tableName := match[1]
			body := match[2]
			if !strings.HasPrefix(tableName, "tb_") {
				t.Fatalf("%s creates table %s without tb_ prefix", entry.Name(), tableName)
			}
			for _, required := range []string{"`status` TINYINT", "`create_time` DATETIME", "`update_time` DATETIME"} {
				if !strings.Contains(body, required) {
					t.Fatalf("%s table %s missing %s", entry.Name(), tableName, required)
				}
			}
		}
	}
}

func TestCaseUIDIsStringCompatible(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "`uid` VARCHAR(128) NOT NULL DEFAULT ''") {
		t.Fatal("tb_troubleshoot_case.uid must stay VARCHAR(128) for string uid compatibility")
	}
}
