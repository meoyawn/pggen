package syntax

import (
	"testing"

	"github.com/meoyawn/pggen/internal/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuerier(t *testing.T) {
	conn, cleanup := pgtest.NewPostgresSchema(t, []string{"schema.sql"})
	defer cleanup()
	require.NoError(t, RegisterTypes(t.Context(), conn))
	q := NewQuerier(conn)
	ctx := t.Context()

	val, err := q.Backtick(ctx)
	assert.NoError(t, err, "Backtick")
	assert.Equal(t, "`", val, "Backtick")

	val, err = q.BacktickDoubleQuote(ctx)
	assert.NoError(t, err, "BacktickDoubleQuote")
	assert.Equal(t, "`\"", val, "BacktickDoubleQuote")

	val, err = q.BacktickQuoteBacktick(ctx)
	assert.NoError(t, err, "BacktickQuoteBacktick")
	assert.Equal(t, "`\"`", val, "BacktickQuoteBacktick")

	val, err = q.BacktickNewline(ctx)
	assert.NoError(t, err, "BacktickNewline")
	assert.Equal(t, "`\n", val, "BacktickNewline")

	val, err = q.BacktickBackslashN(ctx)
	assert.NoError(t, err, "BacktickBackslashN")
	assert.Equal(t, "`\\n", val, "BacktickBackslashN")

	enumVal, err := q.BadEnumName(ctx)
	assert.NoError(t, err, "BadEnumName")
	assert.Equal(t, UnnamedEnum123InconvertibleEnumName, enumVal, "BadEnumName")
}
