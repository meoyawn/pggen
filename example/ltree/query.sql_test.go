package ltree

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jschaf/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()

	q := NewQuerier(conn)
	ctx := t.Context()

	if _, err := q.InsertSampleData(ctx); err != nil {
		t.Fatal(err)
	}

	{
		rows, err := q.FindTopScienceChildren(ctx)
		require.NoError(t, err)
		want := []pgtype.Text{
			{String: "Top.Science", Valid: true},
			{String: "Top.Science.Astronomy", Valid: true},
			{String: "Top.Science.Astronomy.Astrophysics", Valid: true},
			{String: "Top.Science.Astronomy.Cosmology", Valid: true},
		}
		assert.Equal(t, want, rows)
	}

	{
		rows, err := q.FindTopScienceChildrenAgg(ctx)
		require.NoError(t, err)
		want := []string{
			"Top.Science",
			"Top.Science.Astronomy",
			"Top.Science.Astronomy.Astrophysics",
			"Top.Science.Astronomy.Cosmology",
		}
		assert.Equal(t, want, rows)
	}

	{
		in1 := pgtype.Text{String: "foo", Valid: true}
		in2 := []string{"qux", "qux"}
		rows, err := q.FindLtreeInput(ctx, in1, in2)
		require.NoError(t, err)
		assert.Equal(t, FindLtreeInputRow{
			Ltree:   in1,
			TextArr: in2,
		}, rows)
	}
}
