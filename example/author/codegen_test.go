package author

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meoyawn/pggen"
	"github.com/meoyawn/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
)

func TestGenerate_Go_Example_Author(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	err := pggen.Generate(
		pggen.GenerateOptions{
			ConnString:       conn.Config().ConnString(),
			QueryFiles:       []string{"query.sql"},
			OutputDir:        tmpDir,
			GoPackage:        "author",
			Language:         pggen.LangGo,
			InlineParamCount: 2,
		})
	if err != nil {
		t.Fatalf("Generate() example/author: %s", err)
	}

	wantQueryFile := "query.sql.go"
	gotQueryFile := filepath.Join(tmpDir, "query.sql.go")
	assert.FileExists(t, gotQueryFile,
		"Generate() should emit query.sql.go")
	wantQueries, err := os.ReadFile(wantQueryFile)
	if err != nil {
		t.Fatalf("read wanted query.go.sql: %s", err)
	}
	gotQueries, err := os.ReadFile(gotQueryFile)
	if err != nil {
		t.Fatalf("read generated query.go.sql: %s", err)
	}
	assert.Equalf(t, string(wantQueries), string(gotQueries),
		"Got file %s; does not match contents of %s",
		gotQueryFile, wantQueryFile)
}

func TestGenerate_Go_Example_Author_SharedRowAnnotation(t *testing.T) {
	conn, cleanupFunc := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanupFunc()

	tmpDir := t.TempDir()
	queryFile := filepath.Join(tmpDir, "query.sql")
	err := os.WriteFile(queryFile, []byte(`-- FindAuthorById finds one author by ID.
-- name: FindAuthorByID :one row=Author
SELECT * FROM author WHERE author_id = pggen.arg('AuthorID');

-- FindAuthors finds authors by first name.
-- name: FindAuthors :many row=Author
SELECT * FROM author WHERE first_name = pggen.arg('FirstName');
`), 0o600)
	if err != nil {
		t.Fatalf("write query file: %s", err)
	}

	outDir := t.TempDir()
	err = pggen.Generate(
		pggen.GenerateOptions{
			ConnString:       conn.Config().ConnString(),
			QueryFiles:       []string{queryFile},
			OutputDir:        outDir,
			GoPackage:        "author",
			Language:         pggen.LangGo,
			InlineParamCount: 2,
		})
	if err != nil {
		t.Fatalf("Generate() shared row annotation: %s", err)
	}

	gotQueryFile := filepath.Join(outDir, "query.sql.go")
	gotQueries, err := os.ReadFile(gotQueryFile)
	if err != nil {
		t.Fatalf("read generated query.go.sql: %s", err)
	}
	got := string(gotQueries)

	assert.Contains(t, got, "type AuthorRow struct {")
	assert.Contains(t, got, "FindAuthorByID(ctx context.Context, authorID int32) (AuthorRow, error)")
	assert.Contains(t, got, "FindAuthors(ctx context.Context, firstName string) ([]AuthorRow, error)")
	assert.Contains(t, got, "pgx.RowToStructByName[AuthorRow]")
	assert.Equal(t, 1, strings.Count(got, "type AuthorRow struct {"))
	assert.NotContains(t, got, "type FindAuthorByIDRow struct {")
	assert.NotContains(t, got, "type FindAuthorsRow struct {")
}
