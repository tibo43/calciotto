package database

import (
	"strings"
	"testing"
)

func TestInitLocalPerfDBRejectsUnsafeConfiguration(t *testing.T) {
	for _, tc := range []struct {
		key, value, message string
	}{
		{"DATABASE_URL", "configured", "unset DATABASE_URL"},
		{"DATABASE_URL_UNPOOLED", "configured", "unset DATABASE_URL"},
		{"DB_HOST", "example.invalid", "requires a loopback IP"},
		{"DB_NAME", "calciotto", "requires a loopback IP"},
		{"DB_PORT", "invalid", "invalid local DB_PORT"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "")
			t.Setenv("DATABASE_URL_UNPOOLED", "")
			t.Setenv("DB_HOST", "127.0.0.1")
			t.Setenv("DB_NAME", "calciotto_perf")
			t.Setenv(tc.key, tc.value)

			if _, err := InitLocalPerfDB(); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("InitLocalPerfDB returned %v, want a configuration error for %s", err, tc.key)
			}
		})
	}
}
