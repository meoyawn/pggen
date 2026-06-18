package golang

import (
	"testing"

	"github.com/meoyawn/pggen/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatedQuery_EmitRowStructUsesSharedRowType(t *testing.T) {
	query := TemplatedQuery{
		Name:          "FindAuthors",
		RowStructType: "AuthorRow",
		ShouldEmitRow: true,
		ResultKind:    ast.ResultKindMany,
		Outputs: []TemplatedColumn{
			{PgName: "author_id", UpperName: "AuthorID", QualType: "int"},
			{PgName: "name", UpperName: "Name", QualType: "string"},
		},
	}

	assert.Contains(t, query.EmitRowStruct(), "type AuthorRow struct {")
}

func TestAssignSharedRowStructs(t *testing.T) {
	files := []TemplatedFile{
		{
			Queries: []TemplatedQuery{
				{
					Name:          "FindAuthorByID",
					RowType:       "Author",
					RowStructType: "FindAuthorByIDRow",
					ShouldEmitRow: true,
					Outputs: []TemplatedColumn{
						{PgName: "author_id", UpperName: "AuthorID", QualType: "int32"},
						{PgName: "name", UpperName: "Name", QualType: "string"},
					},
				},
				{
					Name:          "FindAuthors",
					RowType:       "Author",
					RowStructType: "FindAuthorsRow",
					ShouldEmitRow: true,
					Outputs: []TemplatedColumn{
						{PgName: "author_id", UpperName: "AuthorID", QualType: "int32"},
						{PgName: "name", UpperName: "Name", QualType: "string"},
					},
				},
			},
		},
	}

	require.NoError(t, assignSharedRowStructs(files))
	assert.Equal(t, "AuthorRow", files[0].Queries[0].RowStructType)
	assert.True(t, files[0].Queries[0].ShouldEmitRow)
	assert.Equal(t, "AuthorRow", files[0].Queries[1].RowStructType)
	assert.False(t, files[0].Queries[1].ShouldEmitRow)
}

func TestAssignSharedRowStructsRejectsIncompatibleShapes(t *testing.T) {
	files := []TemplatedFile{
		{
			Queries: []TemplatedQuery{
				{
					Name:    "FindAuthorByID",
					RowType: "Author",
					Outputs: []TemplatedColumn{
						{PgName: "author_id", UpperName: "AuthorID", QualType: "int32"},
						{PgName: "name", UpperName: "Name", QualType: "string"},
					},
				},
				{
					Name:    "FindAuthors",
					RowType: "Author",
					Outputs: []TemplatedColumn{
						{PgName: "author_id", UpperName: "AuthorID", QualType: "int32"},
						{PgName: "name", UpperName: "Name", QualType: "*string", Nullable: true},
					},
				},
			},
		},
	}

	err := assignSharedRowStructs(files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row=Author used by incompatible queries FindAuthorByID and FindAuthors")
	assert.Contains(t, err.Error(), `pg="name" field=Name type=string nullable=false`)
	assert.Contains(t, err.Error(), `pg="name" field=Name type=*string nullable=true`)
}
