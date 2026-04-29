# ent — Go Entity Framework

Reference doc for using [ent](https://entgo.io) in this project. Sourced via context7 from `entgo.io` (High reputation, 1352 snippets, score 92.65).

ent is a code-generated ORM/entity framework for Go. Schemas are written as Go structs; running `go generate ./ent` produces typed clients, query builders, and migration assets.

## Why ent here

- Replaces the `// TODO wrapped database client` in `internal/data/data.go`.
- Static typing on schema, queries, and edges — fewer runtime SQL errors.
- Auto-migration + Atlas versioned migrations both supported.
- Works with MySQL (current `configs/config.yaml`) and Postgres if we switch.

## Install

```bash
go install entgo.io/ent/cmd/ent@latest
```

Add to `Makefile init` target alongside other code-gen tools.

## 1. Bootstrap a schema

```bash
go run -mod=mod entgo.io/ent/cmd/ent new User
```

Generates `ent/schema/user.go`:

```go
package schema

import "entgo.io/ent"

type User struct {
    ent.Schema
}

func (User) Fields() []ent.Field { return nil }
func (User) Edges()  []ent.Edge  { return nil }
```

## 2. Define fields

```go
import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
)

func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("name"),
        field.String("email").Unique(),
        field.String("title").Optional(),
    }
}
```

Common modifiers: `.Unique()`, `.Optional()`, `.Default(...)`, `.Immutable()`, `.Sensitive()`, `.MaxLen(n)`, `.Match(regex)`.

## 3. Define edges (relations)

### One-to-many: User → Pets

```go
// ent/schema/user.go
import "entgo.io/ent/schema/edge"

func (User) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("pets", Pet.Type),
    }
}
```

### Inverse on Pet (many-to-one)

```go
// ent/schema/pet.go
func (Pet) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("owner", User.Type).
            Ref("pets").
            Unique(),
    }
}
```

`edge.To` = forward, `edge.From(...).Ref(...)` = inverse. `.Unique()` on inverse → many-to-one. Self-edges allowed: `edge.To("friends", Pet.Type)`.

## 4. Indexes

```go
import "entgo.io/ent/schema/index"

func (User) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("field1", "field2"),                    // non-unique
        index.Fields("first_name", "last_name").Unique(),    // composite unique
    }
}
```

## 5. Generate code

```bash
go generate ./ent
```

Run after every schema change. Produces `ent/client.go`, typed builders (`ent.User`, `client.User.Query()`), predicates package (`ent/user`), and migration assets.

Add to `Makefile generate` target: `go generate ./ent`.

## 6. Open client

### MySQL (matches current config)

```go
import (
    "context"
    "log"
    "greeter/ent"
    _ "github.com/go-sql-driver/mysql"
)

client, err := ent.Open("mysql", "root:root@tcp(127.0.0.1:3306)/test?parseTime=True")
if err != nil { log.Fatalf("mysql open: %v", err) }
defer client.Close()

if err := client.Schema.Create(context.Background()); err != nil {
    log.Fatalf("schema create: %v", err)
}
```

### Postgres / SQLite variants

```go
ent.Open("postgres", "host=<h> port=<p> user=<u> dbname=<db> password=<pw>")
ent.Open("sqlite3",  "file:ent?mode=memory&cache=shared&_fk=1")
```

`client.Schema.Create(ctx)` runs **auto-migration** — fine for dev, prefer Atlas versioned for prod (see §10).

## 7. CRUD

### Create

```go
u, err := client.User.
    Create().
    SetName("a8m").
    SetAge(30).
    Save(ctx)
```

### Query with predicates

```go
import "greeter/ent/user"

users, err := client.User.
    Query().
    Where(user.NameEQ("a8m")).
    All(ctx)
```

### Edge predicates

```go
import "greeter/ent/pet"

// pets that have an owner
client.Pet.Query().Where(pet.HasOwner()).All(ctx)

// pets whose owner name = "a8m"
client.Pet.Query().
    Where(pet.HasOwnerWith(user.NameEQ("a8m"))).
    All(ctx)
```

### Boolean composition

```go
client.Todo.Query().
    Where(
        todo.And(
            todo.StatusEQ(todo.StatusCompleted),
            todo.TextHasPrefix("GraphQL"),
        ),
    ).All(ctx)
```

`todo.Or(...)`, `todo.Not(...)` also generated.

## 8. Transactions

```go
tx, err := client.Tx(ctx)
if err != nil { return err }
if err := tx.User.Create().Exec(ctx); err != nil {
    _ = tx.Rollback()
    return err
}
return tx.Commit()
```

Helper pattern: `WithTx(ctx, client, func(tx *ent.Tx) error { ... })`.

## 9. Raw SQL escape hatch

Bypasses hooks/privacy. Use sparingly.

```go
if _, err := client.ExecContext(ctx, "TRUNCATE t1"); err != nil {
    return err
}

tx, _ := client.Tx(ctx)
_, _ = tx.ExecContext(ctx, "SAVEPOINT user_created")
```

## 10. Versioned migrations (Atlas)

Recommended for prod over `Schema.Create`.

```bash
atlas migrate diff add_user_title \
  --dir "file://ent/migrate/migrations" \
  --to "ent://ent/schema" \
  --dev-url "docker://mysql/8/ent"
```

Per dialect:
- MySQL: `docker://mysql/8/ent`
- MariaDB: `docker://mariadb/latest/test`
- Postgres: `docker://postgres/15/test?search_path=public`
- SQLite: `sqlite://file?mode=memory&_fk=1`

Apply with `atlas migrate apply --dir file://ent/migrate/migrations --url <db-url>`.

### Diff/Apply hooks

Intercept Atlas changes, e.g. convert drop+add into rename:

```go
err := client.Schema.Create(ctx,
    schema.WithDiffHook(func(next schema.Differ) schema.Differ {
        return schema.DiffFunc(func(cur, des *atlas.Schema) ([]atlas.Change, error) {
            changes, err := next.Diff(cur, des)
            if err != nil { return nil, err }
            // filter / mutate changes
            return changes, nil
        })
    }),
    schema.WithApplyHook(func(next schema.Applier) schema.Applier {
        return schema.ApplyFunc(func(ctx context.Context, conn dialect.ExecQuerier, plan *migrate.Plan) error {
            return next.Apply(ctx, conn, plan)
        })
    }),
)
```

Rename hook idiom: detect paired `DropColumn("old")` + `AddColumn("new")` in `ModifyTable.Changes` → replace with `&atlas.RenameColumn{From: ..., To: ...}`.

## 11. Project integration plan (kratos layered)

| Layer | Change |
|-------|--------|
| `ent/schema/*` | Schema files (new `ent` dir at repo root) |
| `Makefile` | Add `ent` target: `go generate ./ent` and `init` install for `entgo.io/ent/cmd/ent` |
| `internal/conf/conf.proto` | Already has `Data.database.driver/source` — reuse |
| `internal/data/data.go` | Replace TODO struct with `*ent.Client`; open in `NewData`, return cleanup that calls `client.Close()` |
| `internal/data/<entity>.go` | Implement biz repo interfaces using `client.<Entity>...` builders |
| `internal/biz/<entity>.go` | Define repo interfaces; biz code stays ent-agnostic |
| `cmd/greeter/wire.go` | No change — `*conf.Data` already injected; `NewData` constructor signature stays |

`internal/data/data.go` shape after wiring:

```go
type Data struct {
    db *ent.Client
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
    client, err := ent.Open(c.Database.Driver, c.Database.Source)
    if err != nil { return nil, nil, err }
    cleanup := func() { _ = client.Close() }
    return &Data{db: client}, cleanup, nil
}
```

Repo example mapping biz interface to ent:

```go
type greeterRepo struct{ data *Data }

func (r *greeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
    out, err := r.data.db.Greeter.Create().SetHello(g.Hello).Save(ctx)
    if err != nil { return nil, err }
    return &biz.Greeter{Hello: out.Hello}, nil
}
```

## 12. Useful next steps

- Add `Hooks()` for write-time validation (e.g. clamp a numeric field to a valid range).

## Sources

- [Ent — entgo.io docs (context7)](https://entgo.io/docs)
- [Getting Started](https://entgo.io/docs/getting-started)
- [Schema Edges](https://entgo.io/docs/schema-edges)
- [Schema Indexes](https://entgo.io/docs/schema-indexes)
- [Predicates](https://entgo.io/docs/predicates)
- [CRUD](https://entgo.io/docs/crud)
- [Versioned Migrations](https://entgo.io/docs/versioned/new-migration)
- [Migrate (Atlas hooks)](https://entgo.io/docs/migrate)
- [Feature Flags (raw SQL)](https://entgo.io/docs/feature-flags)
