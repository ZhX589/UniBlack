package db

import (
	"github.com/jackc/pgx/v5"
	"testing"
)

// Exercise the actual PostgreSQL driver's parser without connecting to a DB.
// Complete DSNs must not be rewritten using query-string escaping rules.
func TestPostgresDSNCredentialSemantics(t *testing.T) {
	cases := []struct{ name, dsn, user, password, host, database, application string }{
		{"space", "postgres://u:pass%20word@db:5432/app", "u", "pass word", "db", "app", ""},
		{"plus", "postgres://u:pass+word@db:5432/app", "u", "pass+word", "db", "app", ""},
		{"encoded plus", "postgres://u:pass%2Bword@db:5432/app", "u", "pass+word", "db", "app", ""},
		{"percent", "postgres://u:pass%25word@db:5432/app", "u", "pass%word", "db", "app", ""},
		{"reserved", "postgres://u:p%40%23%3F%3A%2F@db:5432/app", "u", "p@#?:/", "db", "app", ""},
		{"username", "postgres://us%40er:pw@db:5432/app", "us@er", "pw", "db", "app", ""},
		{"IPv6", "postgres://u:pw@[::1]:5432/app", "u", "pw", "::1", "app", ""},
		{"query at sign", "postgres://u:p@db:5432/app?application_name=a@b", "u", "p", "db", "app", "a@b"},
		{"colon", "postgres://u:p:word@db:5432/app", "u", "p:word", "db", "app", ""},
		{"keyword with URL password", "host=db port=5432 user=u password='https://a@b' dbname=app", "u", "https://a@b", "db", "app", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := pgx.ParseConfig(tc.dsn)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.User != tc.user || cfg.Password != tc.password || cfg.Host != tc.host || cfg.Port != 5432 || cfg.Database != tc.database {
				t.Fatal("driver changed connection field semantics")
			}
			if tc.application != "" && cfg.RuntimeParams["application_name"] != tc.application {
				t.Fatal("query parameter corrupted")
			}
		})
	}
}
