[![Test](https://github.com/meoyawn/pggen/workflows/Test/badge.svg)](https://github.com/meoyawn/pggen/actions?query=workflow%3ATest)
[![Lint](https://github.com/meoyawn/pggen/workflows/Lint/badge.svg)](https://github.com/meoyawn/pggen/actions?query=workflow%3ALint)
[![GoReportCard](https://goreportcard.com/badge/github.com/meoyawn/pggen)](https://goreportcard.com/report/github.com/meoyawn/pggen)

# pggen - generate type safe Go methods from Postgres SQL queries

pggen generates Go code to provide a typesafe wrapper to run Postgres queries.
If Postgres can run the query, pggen can generate code for it. The generated 
code is strongly-typed with rich mappings between Postgres types and Go types
without relying on `interface{}`. pggen uses prepared queries, so you don't 
have to worry about SQL injection attacks. 

How to use pggen in three steps:

1.  Write arbitrarily complex SQL queries with a name and a `:one`, `:many`,
    `:stream`, or `:exec` annotation. Declare inputs with
    `pggen.arg('input_name')`.

    ```sql
    -- name: SearchScreenshots :many
    SELECT ss.id, array_agg(bl) AS blocks
    FROM screenshots ss
      JOIN blocks bl ON bl.screenshot_id = ss.id
    WHERE bl.body LIKE pggen.arg('body') || '%'
    GROUP BY ss.id
    ORDER BY ss.id
    LIMIT pggen.arg('limit') OFFSET pggen.arg('offset');
    ```

2.  Run pggen to generate Go code to create type-safe methods for each query.
   
    ```bash
    pggen gen go \
        --schema-glob schema.sql \
        --query-glob 'screenshots/*.sql' \
        --go-type 'int8=int' \
        --go-type 'text=string'
    ```
    
    That command generates methods and type definitions like below. The full
    example is in [./example/composite/query.sql.go].
    
    ```go
    type SearchScreenshotsParams struct {
        Body   string
        Limit  int
        Offset int
    }

    type SearchScreenshotsRow struct {
        ID     int      `json:"id"`
        Blocks []Blocks `json:"blocks"`
    }
    
    // Blocks represents the Postgres composite type "blocks".
    type Blocks struct {
        ID           int    `json:"id"`
        ScreenshotID int    `json:"screenshot_id"`
        Body         string `json:"body"`
    }
    
    func (q *DBQuerier) SearchScreenshots(
        ctx context.Context,
        params SearchScreenshotsParams,
    ) ([]SearchScreenshotsRow, error) {
        /* omitted */
    }

    func (q *DBQuerier) StreamScreenshots(
        ctx context.Context,
        params StreamScreenshotsParams,
        yield func(StreamScreenshotsRow) error,
    ) error {
        /* omitted */
    }
    ```
    
3.  Use the generated code.

    ```go
    var conn *pgx.Conn
	q := NewQuerier(conn)
    rows, err := q.SearchScreenshots(ctx, SearchScreenshotsParams{
        Body:   "some_prefix",
        Limit:  50,
        Offset: 200,
    })
    ```
[./example/composite/query.sql.go]: ./example/composite/query.sql.go

## Pitch

Why should you use `pggen` instead of the [myriad] of Go SQL bindings?

- pggen generates code by introspecting the database system catalogs, so you 
  can use *any* database extensions or custom methods, and it will just work.
  For database types that pggen doesn't recognize, you can provide your own
  type mappings.

- pggen scales to Postgres databases of any size and supports incremental 
  adoption. pggen is narrowly tailored to only generate code for queries you 
  write in SQL. pggen will not create a model for every database object. 
  Instead, pggen only generates structs necessary to run the queries you 
  specify.

- pggen works with any Postgres database with any extensions. Under the hood, 
  pggen runs each query and uses the Postgres catalog tables, `pg_type`, 
  `pg_class`, and `pg_attribute`, to get **perfect type information** for both 
  the query parameters and result columns.
  
- pggen works with all Postgres queries. If Postgres can run the query, pggen
  can generate Go code for the query.
  
- pggen uses [pgx], a faster replacement for [lib/pq], the original Go Postgres
  library that's now in maintenance mode.

- pggen-generated queriers work with pgx connection transports like
  [`*pgx.Conn`], [`pgx.Tx`], and [`*pgxpool.Pool`].
  
[pgx]: https://github.com/jackc/pgx
[lib/pq]: https://github.com/lib/pq

## Anti-pitch

I'd like to try to convince you why you *shouldn't* use pggen. Often, this
is far more revealing than the pitch.

- You want auto-generated models for every table in your database. pggen only
  generates code for each query in a query file. pggen requires custom SQL for
  even the simplest CRUD queries. Use [gorm] or any of alternatives listed
  at [awesome Go ORMs].

- You use a database other than Postgres. pggen only supports Postgres. [sqlc],
  a similar tool which inspired pggen, has early support for MySQL.

- You want an active-record pattern where models have methods like `find`, 
  `create`, `update`, and `delete`. pggen only generates code for queries you 
  write. Use [gorm].
  
- You prefer building queries in a Go dialect instead of SQL. I'd recommend 
  investing in really learning SQL; it will payoff. Otherwise, use 
  [squirrel], [goqu], or [go-sqlbuilder]
  
- You don't want to add a Postgres or Docker dependency to your build phase.
  Use [sqlc], though you might still need Docker. sqlc generates code by parsing
  the schema file and queries in Go without using Postgres.

[myriad]: https://github.com/d-tsuji/awesome-go-orms
[sqlc]: https://github.com/kyleconroy/sqlc
[gorm]: https://gorm.io/index.html
[squirrel]: https://github.com/Masterminds/squirrel
[goqu]: https://github.com/doug-martin/goqu
[go-sqlbuilder]: https://github.com/huandu/go-sqlbuilder
[awesome Go ORMs]: https://github.com/d-tsuji/awesome-go-orms

# Install

### Download precompiled binaries

Precompiled binaries from the latest release. Change `~/bin` if you want to
install to a different directory. All assets are listed on the [releases] page.

[releases]: https://github.com/meoyawn/pggen/releases

-   MacOS Apple Silicon (arm64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/meoyawn/pggen/releases/latest/download/pggen-darwin-arm64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```
    
-   MacOS Intel (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/meoyawn/pggen/releases/latest/download/pggen-darwin-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```

-   Linux (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/meoyawn/pggen/releases/latest/download/pggen-linux-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```
    
-   Windows (amd64)

    ```shell
    mkdir -p ~/bin \
      && curl --silent --show-error --location --fail 'https://github.com/meoyawn/pggen/releases/latest/download/pggen-windows-amd64.tar.xz' \
      | tar -xJf - -C ~/bin/    
    ```

Make sure pggen works:

```bash
pggen gen go --help
```

### Install from source

Requires Go 1.16 because pggen uses `go:embed`. Installs to `$GOPATH/bin`.

```shell
go install github.com/meoyawn/pggen/cmd/pggen@latest
```
    
Make sure pggen works:

```bash
pggen gen go --help
```

## Usage

Generate code using Docker to create the Postgres database from a schema file:

```bash
# --schema-glob runs all matching files on Dockerized Postgres during database 
# creation.
pggen gen go \
    --schema-glob author/schema.sql \
    --query-glob author/query.sql

# Output: author/query.go.sql

# Or with multiple schema files. The schema files run on Postgres
# in the order they appear on the command line.
pggen gen go \
    --schema-glob author/schema.sql \
    --schema-glob book/schema.sql \
    --schema-glob publisher/schema.sql \
    --query-glob author/query.sql

# Output: author/query.sql.go
```

Generate code using an existing Postgres database (useful for custom setups):

```bash
pggen gen go \
    --query-glob author/query.sql \
    --postgres-connection "user=postgres port=5555 dbname=pggen"

# Output: author/query.sql.go
```

Generate code for multiple query files. All the query files must reside in
the same directory. If query files reside in different directories, you can use
`--output-dir` to set a single output directory:

```bash
pggen gen go \
    --schema-glob schema.sql \
    --query-glob author/fiction.sql \
    --query-glob author/nonfiction.sql \
    --query-glob author/bestselling.sql

# Output: author/fiction.sql.go
#         author/nonfiction.sql.go
#         author/bestselling.sql.go

# Or, using a glob. Notice quotes around glob pattern to prevent shell 
# expansion.
pggen gen go \
    --schema-glob schema.sql \
    --query-glob 'author/*.sql'
```

# Examples

Examples embedded in the repo:

- [./example/acceptance_test.go] - End-to-end examples of how to call pggen.
- [./example/author] - A single table schema with simple queries.
- [./example/composite] - Arrays of composite (aka row or table) types.
- [./example/custom_types] - Mapping new Postgres types to Go types.
- [./example/device] - Complex queries with a 1:many relationship between a 
  `user` table and `device` table.
- [./example/enums] - Postgres and Go enums.
- [./example/erp] - A few tables with mildly complex queries.
- [./example/go_pointer_types] - Mapping to pointer types like `*int` instead
  of `pgtype.Int8`.
- [./example/ltree] - Support for the ltree Postgres extension.
- [./example/nested] - Complex, nested composite (aka row or table) types.
- [./example/pgcrypto] - pgcrypto Postgres extension.
- [./example/syntax] - A smoke test of interesting SQL syntax.
- [./example/void] - Support for void in select columns.

[./example/acceptance_test.go]: ./example/acceptance_test.go
[./example/author]: ./example/author
[./example/composite]: ./example/composite
[./example/custom_types]: ./example/custom_types
[./example/device]: ./example/device
[./example/enums]: ./example/enums
[./example/erp]: ./example/erp
[./example/go_pointer_types]: ./example/go_pointer_types
[./example/ltree]: ./example/ltree
[./example/nested]: ./example/nested
[./example/syntax]: ./example/syntax
[./example/pgcrypto]: ./example/pgcrypto
[./example/void]: ./example/void

# Features

-   **JSON struct tags**: All `<query_name>Row` structs include JSON struct tags
    using the Postgres column name. To change the struct tag, use an SQL column 
    alias.
  
    ```sql
    -- name: FindAuthors :many
    SELECT first_name, last_name as family_name FROM author;
    ```
    
    Generates:
    
    ```go
    type FindAuthorsRow struct {
        FirstName   string `json:"first_name"`
        FamilyName  string `json:"family_name"`
    }
    ```

-   **Acronyms**: Custom acronym support so that `author_id` renders as 
    `AuthorID` instead of `AuthorId`. Supports two formats:
    
    1. Long form: `--acronym <word>=<relacement>`: replaces `<word>` with 
       `<replacement>` literally. Useful for plural acronyms like `author_ids` 
       which should render as `AuthorIDs`, not `AuthorIds`. For the IDs example,
        use `--acronym ids=IDs`.
       
    2. Short form: `--acronym <word>`: replaces `<word>` with uppercase 
       `<WORD>`. Equivalent to `--acronym <word>=<WORD>`
       
    By default, pggen includes `--acronym id` to render `id` as `ID`.

-   **Enums**: Postgres enums map to Go string constant enums. The Postgres 
    type:
    
    ```sql
    CREATE TYPE device_type AS ENUM ('undefined', 'phone', 'ipad');
    ```
    
    pggen generates the following Go code when used in a query:
    
    ```go
    // DeviceType represents the Postgres enum device_type.
    type DeviceType string

    const (
        DeviceTypeUndefined DeviceType = "undefined"
        DeviceTypePhone     DeviceType = "phone"
        DeviceTypeIpad      DeviceType = "ipad"
    )

    func (d DeviceType) String() string { return string(d) }
    ```

-   **Custom types**: Use a custom Go type to represent a Postgres type with the 
    `--go-type` flag. The format is `<pg_type>=<qualified_go_type>`. For 
    example:

    ```sh
    pggen gen go \
        --schema-glob example/custom_types/schema.sql \
        --query-glob example/custom_types/query.sql \
        --go-type 'int8=*int' \
        --go-type 'int4=int' \
        --go-type '_int4=[]int' \
        --go-type 'text=*github.com/meoyawn/pggen/mytype.String' \
        --go-type '_text=[]*github.com/meoyawn/pggen/mytype.String'
    ```
    
    pggen only changes the generated Go signatures and scan destinations. pgx
    must still be able to decode the Postgres type using the given Go type.
    That means the Go type must fulfill at least one of the following:
    
    - The Go type is a wrapper around primitive type, like `type AuthorID int`.
      pgx will use decode methods on the underlying primitive type.

    - The Go type implements the scanner interfaces supported by the
      [`pgtype.Codec`] for that Postgres type. See the [pgtype package] for
      built-in codecs and interfaces.

    - The Go type implements [`sql.Scanner`]. Query parameters can also
      implement [`driver.Valuer`].

    - The pgx connection executing the query has registered the Postgres type
      in its pgx v5 type map. For enum, composite, and array types that
      pgx can inspect, pggen generates `RegisterTypes(ctx, *pgx.Conn)`, which
      calls [`pgx.Conn.LoadType`] and [`pgtype.Map.RegisterType`].

      ```go
      conn, err := pgx.Connect(ctx, url)
      if err != nil {
          return err
      }
      if err := RegisterTypes(ctx, conn); err != nil {
          return err
      }

      q := NewQuerier(conn)
      ```

      If you use [`*pgxpool.Pool`], call `RegisterTypes` from
      [`pgxpool.Config.AfterConnect`] so every pooled connection has the same
      type map.

    - The pgx connection has a custom [`pgtype.Type`] registered with a
      [`pgtype.Codec`]. This is useful for user-defined base types and extension
      types that need application-provided codecs. See the
      [example/custom_types test] for an example.

      ```go
      conn.TypeMap().RegisterType(&pgtype.Type{
          Name:  "my_int",
          OID:   myIntOID,
          Codec: pgtype.Int2Codec{},
      })
      ```
    
    - pgx is able to use reflection to build an object to write fields into.

-   **Nested structs (composite types)**: pggen creates child structs to 
    represent Postgres [composite types] that appear in output columns.

    ```sql
    -- name: FindCompositeUser :one
    SELECT ROW (15, 'qux')::"user" AS "user";
    ```
    
    pggen generates the following Go code:
    
    ```go
    // User represents the Postgres composite type "user".
    type User struct {
        ID   pgtype.Int8
        Name pgtype.Text
    }
    
    func (q *DBQuerier) FindCompositeUser(ctx context.Context) (User, error) {}
    ```

[pgtype package]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype
[`pgtype.Codec`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#Codec
[`pgtype.Type`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#Type
[`pgtype.Map.RegisterType`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#Map.RegisterType
[`pgx.Conn.LoadType`]: https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.LoadType
[`sql.Scanner`]: https://golang.org/pkg/database/sql/#Scanner
[`driver.Valuer`]: https://pkg.go.dev/database/sql/driver#Valuer
[composite types]: https://www.postgresql.org/docs/current/rowtypes.html
[example/custom_types test]: ./example/custom_types/query.sql_test.go

# IDE integration

If your IDE provides SQL autocomplete, you may want to get rid of its warnings
by declaring the following DDL schema.

```sql
-- Exists solely so editors don't underline every pggen.arg() expression in
-- squiggly red.
CREATE SCHEMA pggen;

-- pggen.arg defines a named parameter that's eventually compiled into a
-- placeholder for a prepared query: $1, $2, etc.
CREATE FUNCTION pggen.arg(param TEXT) RETURNS text AS $$SELECT null$$ LANGUAGE sql;
```

# Tutorial

Let's say we have a database with the following schema in `author/schema.sql`:

```sql
CREATE TABLE author (
  author_id  serial PRIMARY KEY,
  first_name text NOT NULL,
  last_name  text NOT NULL,
  suffix     text NULL
)
```

First, write a query in the file `author/query.sql`. The query name is 
`FindAuthors` and the query returns `:many` rows. A query can return `:many`
rows, `:one` row, `:stream` rows through a callback, or `:exec` for update,
insert, and delete queries.

```sql
-- FindAuthors finds authors by first name.
-- name: FindAuthors :many
SELECT * FROM author WHERE first_name = pggen.arg('first_name');

-- StreamAuthors streams authors by first name.
-- name: StreamAuthors :stream
SELECT * FROM author WHERE first_name = pggen.arg('first_name');
```

Second, use pggen to generate Go code to `author/query.sql.go`:

```bash
pggen gen go \
    --schema-glob author/schema.sql \
    --query-glob author/query.sql
```

We'll walk through the generated file `author/query.sql.go`:

-   The `Querier` interface defines one method for each SQL query.
  
    ```go
    // Querier is a typesafe Go interface backed by SQL queries.
    type Querier interface {
        // FindAuthors finds authors by first name.
        FindAuthors(ctx context.Context, firstName string) ([]FindAuthorsRow, error)

        // StreamAuthors streams authors by first name.
        StreamAuthors(ctx context.Context, firstName string, yield func(StreamAuthorsRow) error) error
    }
    ```

-   The `DBQuerier` struct implements the `Querier` interface with concrete
    implementations of each query method.

    ```sql
    type DBQuerier struct {
        conn genericConn
    }
    ```

-   Create `DBQuerier` with `NewQuerier`. The `genericConn` parameter is an 
    interface over the different pgx connection transports so that `DBQuerier` 
    doesn't force you to use a specific connection transport. [`*pgx.Conn`], 
    [`pgx.Tx`], and [`*pgxpool.Pool`] all implement `genericConn`.

    ```sql
    // NewQuerier creates a DBQuerier that implements Querier. conn is typically
    // *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
    func NewQuerier(conn genericConn) *DBQuerier {
        return &DBQuerier{
            conn: conn,
        }
    }
    ```

-   For custom PostgreSQL types, pggen generates `RegisterTypes`. Call it once
    for each `*pgx.Conn` before running queries that use generated enum,
    composite, or array types.

    ```go
    // RegisterTypes loads custom PostgreSQL types into conn's pgx type map.
    func RegisterTypes(ctx context.Context, conn *pgx.Conn) error {}
    ```
    
-   pggen embeds the SQL query formatted for a Postgres `PREPARE` statement with
    parameters indicated by `$1`, `$2`, etc. instead of 
    `pggen.arg('first_name')`.

    ```sql
    const findAuthorsSQL = `SELECT * FROM author WHERE first_name = $1;`
    ```
    
-   pggen generates a row struct for each query named `<query_name>Row`.
    pggen transforms the output column names into struct field names from
    `lower_snake_case` to `UpperCamelCase` in [internal/casing/casing.go]. 
    pggen derives JSON struct tags from the Postgres column names. To change the
    JSON struct name, change the column name in the query.
    
    ```sql
    type FindAuthorsRow struct {
        AuthorID  int32   `json:"author_id" db:"author_id"`
        FirstName string  `json:"first_name" db:"first_name"`
        LastName  string  `json:"last_name" db:"last_name"`
        Suffix    *string `json:"suffix" db:"suffix"`
    }
    ```

    As a convenience, if a query only generates a single column, pggen skips
    creating the `<query_name>Row` struct and returns the type directly.  For
    example, the generated query for `SELECT author_id from author` returns 
    `int32`, not a `<query_name>Row` struct.
    
    pggen infers struct field types by preparing the query. When Postgres
    prepares a query, Postgres returns the parameter and column types as OIDs.
    pggen finds the type name from the returned OIDs in
    [internal/codegen/golang/gotype/types.go].
    
    Choosing an appropriate type is more difficult than might seem at first 
    glance due to `null`. When Postgres reports that a column has a type `text`,
    that column can have  both `text` and `null` values. So, the Postgres `text`
    represented in Go can be either a `string` or `nil`. [`pgtype`] provides 
    nullable types for all built-in Postgres types. pggen tries to infer if a 
    column is nullable or non-nullable. If a column is nullable, pggen uses a
    nullable Go type like `*string` or a `pgtype` type. If a column is
    non-nullable, pggen uses a more ergonomic type like `string`. pggen's
    nullability inference implemented in [internal/pginfer/nullability.go] is
    rudimentary; a proper approach requires a full explain-plan with some
    control flow analysis.
    
-   Lastly, pggen generates the implementation for each query.

    As a convenience, if a there are only one or two query parameters, pggen
    inlines the parameters into the method definition, as with `firstName` 
    below. If there are three or more parameters, pggen creates a struct named
    `<query_name>Params` to pass the parameters to the query method.
    
    ```sql
    // FindAuthors implements Querier.FindAuthors.
    func (q *DBQuerier) FindAuthors(ctx context.Context, firstName string) ([]FindAuthorsRow, error) {
        ctx = context.WithValue(ctx, QueryName{}, "FindAuthors")
        rows, err := q.conn.Query(ctx, findAuthorsSQL, firstName)
        if err != nil {
            var zero []FindAuthorsRow
            return zero, fmt.Errorf("query FindAuthors: %w", err)
        }

        result, err := pgx.CollectRows(rows, pgx.RowToStructByName[FindAuthorsRow])
        if err != nil {
            var zero []FindAuthorsRow
            return zero, fmt.Errorf("scan FindAuthors row: %w", err)
        }
        return result, nil
    }

    // StreamAuthors implements Querier.StreamAuthors.
    func (q *DBQuerier) StreamAuthors(ctx context.Context, firstName string, yield func(StreamAuthorsRow) error) error {
        ctx = context.WithValue(ctx, QueryName{}, "StreamAuthors")
        rows, err := q.conn.Query(ctx, streamAuthorsSQL, firstName)
        if err != nil {
            return fmt.Errorf("query StreamAuthors: %w", err)
        }

        var item StreamAuthorsRow
        if err := forEachRow(rows, []any{&item.AuthorID, &item.FirstName, &item.LastName, &item.Suffix}, &item, yield); err != nil {
            return fmt.Errorf("stream StreamAuthors row: %w", err)
        }
        return nil
    }
    ```

[`*pgx.Conn`]: https://pkg.go.dev/github.com/jackc/pgx/v5#Conn
[`pgx.Tx`]: https://pkg.go.dev/github.com/jackc/pgx/v5#Tx
[`*pgxpool.Pool`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Pool
[`pgxpool.Config.AfterConnect`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Config
[internal/casing/casing.go]: ./internal/casing/casing.go
[internal/codegen/golang/gotype/types.go]: ./internal/codegen/golang/gotype/types.go
[`pgtype`]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype
[internal/pginfer/nullability.go]: ./internal/pginfer/nullability.go

# Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [ARCHITECTURE.md](ARCHITECTURE.md).

# Acknowledgments

pggen was directly inspired by [sqlc]. The primary difference between pggen and
sqlc is how each tool infers the type and nullability of the input parameters
and output columns for SQL queries.

sqlc parses the queries in Go code, using Cgo to call the Postgres `parser.c` 
library. After parsing, sqlc infers the types of the query parameters and result
columns using custom logic in Go. In contrast, pggen gets the same type 
information by running the queries on Postgres and then fetching the type 
information for Postgres catalog tables. 

Use sqlc if you don't wish to run Postgres to generate code or if you need
better nullability analysis than pggen provides.

Use pggen if you can run Postgres for code generation, and you use complex 
queries that sqlc is unable to parse. Additionally, use pggen if you have a 
custom database setup that's difficult to replicate in a schema file. pggen
supports running on any database with any extensions.
