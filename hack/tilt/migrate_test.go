package tilt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		missingDirectory bool
		hasMigration     bool
		connectionErrors int
		migrationError   bool
		versionCalls     int
		upCalls          int
		wantError        string
	}{
		{
			name:             "missing directory fails before connecting",
			missingDirectory: true,
			wantError:        "Migration directory does not exist",
		},
		{
			name:         "empty directory initializes version table without running up",
			versionCalls: 1,
		},
		{
			name:             "waits for database before migrating",
			hasMigration:     true,
			connectionErrors: 2,
			versionCalls:     3,
			upCalls:          1,
		},
		{
			name:             "connection failures are bounded and reported",
			hasMigration:     true,
			connectionErrors: 30,
			versionCalls:     30,
			wantError:        "Database did not become ready after 30 attempts:\nconnection refused",
		},
		{
			name:           "migration failure is reported without retrying",
			hasMigration:   true,
			migrationError: true,
			versionCalls:   1,
			upCalls:        1,
			wantError:      "migration failed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			migrationsDir := filepath.Join(dir, "sql migrations")
			if !testCase.missingDirectory {
				require.NoError(t, os.Mkdir(migrationsDir, 0o700))
			}
			if testCase.hasMigration {
				require.NoError(t, os.WriteFile(
					filepath.Join(migrationsDir, "00001_test.sql"),
					[]byte("-- +goose Up\nSELECT 1;\n"),
					0o600,
				))
			}
			// Fake only external commands: exercise the actual migration script,
			// including retry boundaries, environment propagation and exit status.
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(`#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >> "$TEST_LOG"
test "$GOOSE_DRIVER" = postgres
test "$GOOSE_DBSTRING" = 'postgres://test:password@localhost:25432/test'
test "$GOOSE_MIGRATION_DIR" = "$TEST_MIGRATIONS_DIR"
case "$*" in
  'tool goose -timeout 2s version')
    count=0
    if [[ -f "$TEST_COUNT" ]]; then read -r count < "$TEST_COUNT"; fi
    count=$((count + 1))
    printf '%s\n' "$count" > "$TEST_COUNT"
    if ((count <= TEST_CONNECTION_ERRORS)); then
      echo 'connection refused' >&2
      exit 1
    fi
    ;;
  'tool goose up')
    if [[ "$TEST_MIGRATION_ERROR" = true ]]; then
      echo 'migration failed' >&2
      exit 1
    fi
    ;;
  *) exit 2 ;;
esac
`), 0o700))
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, "sleep"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o700,
			))
			logPath := filepath.Join(dir, "commands.log")
			require.NoError(t, os.WriteFile(logPath, nil, 0o600))
			cmd := exec.Command("bash", "migrate.sh")
			cmd.Env = append(os.Environ(),
				"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GOOSE_DRIVER=postgres",
				"GOOSE_DBSTRING=postgres://test:password@localhost:25432/test",
				"GOOSE_MIGRATION_DIR="+migrationsDir,
				"TEST_MIGRATIONS_DIR="+migrationsDir,
				"TEST_LOG="+logPath,
				"TEST_COUNT="+filepath.Join(dir, "count"),
				fmt.Sprintf("TEST_CONNECTION_ERRORS=%d", testCase.connectionErrors),
				fmt.Sprintf("TEST_MIGRATION_ERROR=%t", testCase.migrationError),
			)
			output, err := cmd.CombinedOutput()
			if testCase.wantError != "" {
				require.Error(t, err)
				require.Contains(t, string(output), testCase.wantError)
			} else {
				require.NoError(t, err, string(output))
			}
			commands, err := os.ReadFile(logPath)
			require.NoError(t, err)
			require.Equal(t, testCase.versionCalls, strings.Count(string(commands), " version\n"))
			require.Equal(t, testCase.upCalls, strings.Count(string(commands), " up\n"))
		})
	}
}
