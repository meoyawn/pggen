package pgtest

import (
	"context"
	"math/rand/v2"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CleanupFunc deletes the schema and all database objects.
type CleanupFunc func()

type Option func(config *pgx.ConnConfig)

// NewPostgresSchemaString opens a connection with search_path set to a randomly
// named, new schema and loads the sql string.
func NewPostgresSchemaString(t *testing.T, sql string, opts ...Option) (*pgx.Conn, CleanupFunc) {
	t.Helper()
	// Create a new schema.
	connStr := postgresConnString()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("connect to docker postgres: %s", err)
	}
	schema := "pggen_test_" + strconv.Itoa(int(rand.Int32())) //nolint:gosec
	if _, err = conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create new schema: %s", err)
	}
	t.Logf("created schema: %s", schema)

	// Load SQL files into new schema.
	schemaConnStr := postgresSchemaConnString(connStr, schema)
	connCfg, err := pgx.ParseConfig(schemaConnStr)
	if err != nil {
		t.Fatalf("parse config: %q: %s", schemaConnStr, err)
	}
	for _, opt := range opts {
		opt(connCfg)
	}
	schemaConn, err := pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		t.Fatalf("connect to docker postgres with search path: %s", err)
	}

	if _, err := schemaConn.Exec(ctx, sql); err != nil {
		t.Fatalf("run sql: %s", err)
	}
	if err := registerExtensionTypes(ctx, schemaConn); err != nil {
		t.Fatalf("register extension types: %s", err)
	}
	if err := registerSchemaTypes(ctx, schemaConn); err != nil {
		t.Fatalf("register schema types: %s", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("close conn: %s", err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Errorf("close conn: %s", err)
		}
		if err = schemaConn.Close(ctx); err != nil {
			t.Errorf("close schema conn: %s", err)
		}
	}
	return schemaConn, cleanup
}

func registerExtensionTypes(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, `
SELECT typname, oid
FROM pg_type
WHERE typname IN ('citext')
  AND pg_type_is_visible(oid)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var oid uint32
		if err := rows.Scan(&name, &oid); err != nil {
			return err
		}
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  name,
			OID:   oid,
			Codec: pgtype.TextCodec{},
		})
	}
	return rows.Err()
}

func registerSchemaTypes(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, `
SELECT format('%I.%I', nsp.nspname, typ.typname)
FROM pg_type typ
JOIN pg_namespace nsp ON nsp.oid = typ.typnamespace
WHERE typ.typnamespace = current_schema()::regnamespace
  AND (typ.typtype IN ('c', 'd', 'e') OR (typ.typtype = 'b' AND typ.typelem <> 0 AND typ.typname LIKE '\_%'))
ORDER BY typ.typname`)
	if err != nil {
		return err
	}
	var typeNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		typeNames = append(typeNames, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	pending := typeNames
	for len(pending) > 0 {
		remaining := pending[:0]
		loaded := 0
		for _, name := range pending {
			dt, err := conn.LoadType(ctx, name)
			if err != nil {
				remaining = append(remaining, name)
				continue
			}
			conn.TypeMap().RegisterType(dt)
			loaded++
		}
		if loaded == 0 {
			return nil
		}
		pending = remaining
	}
	return nil
}

func postgresConnString() string {
	if connStr := os.Getenv("PGURL"); connStr != "" {
		return connStr
	}
	return "user=postgres password=hunter2 host=localhost port=5555 dbname=pggen"
}

func postgresSchemaConnString(connStr string, schema string) string {
	if u, err := url.Parse(connStr); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		return u.String()
	}
	return connStr + " search_path=" + schema
}

// NewPostgresSchema opens a connection with search_path set to a randomly
// named, new schema and loads all sqlFiles.
func NewPostgresSchema(t *testing.T, sqlFiles []string, opts ...Option) (*pgx.Conn, CleanupFunc) {
	t.Helper()
	sb := &strings.Builder{}
	for _, file := range sqlFiles {
		bs, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read test db sql file: %s", err)
		}
		sb.Write(bs)
		sb.WriteString(";\n\n -- FILE: ")
		sb.WriteString(file)
		sb.WriteString("\n")

	}
	return NewPostgresSchemaString(t, sb.String(), opts...)
}
