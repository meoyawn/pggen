package function

import (
	"testing"

	"github.com/meoyawn/pggen/internal/difftest"
	"github.com/meoyawn/pggen/internal/ptrs"
	"github.com/stretchr/testify/require"

	"github.com/meoyawn/pggen/internal/pgtest"
)

func TestNewQuerier_OutParams(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	require.NoError(t, RegisterTypes(t.Context(), conn))

	q := NewQuerier(conn)

	t.Run("OutParams", func(t *testing.T) {
		got, err := q.OutParams(t.Context())
		require.NoError(t, err)
		want := []OutParamsRow{
			{
				Items: []ListItem{{Name: ptrs.String("some_name"), Color: ptrs.String("some_color")}},
				Stats: ListStats{
					Val1: ptrs.String("abc"),
					Val2: []*int32{ptrs.Int32(1), ptrs.Int32(2)},
				},
			},
		}
		difftest.AssertSame(t, want, got)
	})
}
