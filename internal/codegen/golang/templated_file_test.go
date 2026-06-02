package golang

import (
	"testing"

	"github.com/meoyawn/pggen/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestResolveProjectionGetterNames(t *testing.T) {
	cols := resolveProjectionGetterNames([]TemplatedColumn{
		{UpperName: "ID"},
		{UpperName: "GetID"},
		{UpperName: "Foo"},
		{UpperName: "FooValue"},
		{UpperName: "GetFoo"},
		{UpperName: "GetFooValue"},
		{UpperName: "Bar"},
		{UpperName: "Bar"},
	})

	got := make([]string, 0, len(cols))
	for _, col := range cols {
		got = append(got, col.GetterName)
	}

	assert.Equal(t, []string{
		"GetIDValue",
		"GetGetID",
		"GetFooValue2",
		"GetFooValueValue",
		"GetGetFoo",
		"GetGetFooValue",
		"GetBar",
		"GetBarValue",
	}, got)
}

func TestTemplatedQuery_EmitProjection(t *testing.T) {
	query := TemplatedQuery{
		Name:       "FindAuthors",
		ResultKind: ast.ResultKindMany,
		Outputs: []TemplatedColumn{
			{UpperName: "AuthorID", GetterName: "GetAuthorID", QualType: "int"},
			{UpperName: "Name", GetterName: "GetName", QualType: "string"},
		},
	}

	assert.Equal(t, `

type FindAuthorsProjection interface {
	GetAuthorID() int
	GetName() string
}`, query.EmitProjectionInterface())

	assert.Equal(t, `

func (r FindAuthorsRow) GetAuthorID() int { return r.AuthorID }

func (r FindAuthorsRow) GetName() string { return r.Name }`, query.EmitRowGetterMethods())
}

func TestTemplatedQuery_EmitProjectionSkipsSingleColumn(t *testing.T) {
	query := TemplatedQuery{
		Name:       "FindAuthorID",
		ResultKind: ast.ResultKindMany,
		Outputs: []TemplatedColumn{
			{UpperName: "AuthorID", GetterName: "GetAuthorID", QualType: "int"},
		},
	}

	assert.Empty(t, query.EmitProjectionInterface())
	assert.Empty(t, query.EmitRowGetterMethods())
}
