package gorm

import (
	"context"
	"database/sql"

	"gossh/gorm/clause"
	"gossh/gorm/schema"
)

// Dialector GORM database dialector
type Dialector interface {
	Name() string
	Initialize(*DB) error
	Migrator(db *DB) Migrator
	DataTypeOf(*schema.Field) string
	DefaultValueOf(*schema.Field) clause.Expression
	BindVarTo(writer clause.Writer, stmt *Statement, v any)
	QuoteTo(clause.Writer, string)
	Explain(sql string, vars ...any) string
}

// Plugin GORM plugin interface
type Plugin interface {
	Name() string
	Initialize(*DB) error
}

type ParamsFilter interface {
	ParamsFilter(ctx context.Context, sql string, params ...any) (string, []any)
}

// ConnPool db conns pool interface
type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SavePointerDialectorInterface save pointer interface
type SavePointerDialectorInterface interface {
	SavePoint(tx *DB, name string) error
	RollbackTo(tx *DB, name string) error
}

// TxBeginner tx beginner
type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// ConnPoolBeginner conn pool beginner
type ConnPoolBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (ConnPool, error)
}

// TxCommitter tx committer
type TxCommitter interface {
	Commit() error
	Rollback() error
}

// Tx sql.Tx interface
type Tx interface {
	ConnPool
	TxCommitter
	StmtContext(ctx context.Context, stmt *sql.Stmt) *sql.Stmt
}

// Valuer gorm valuer interface
type Valuer interface {
	GormValue(context.Context, *DB) clause.Expr
}

// GetDBConnector SQL db connector
type GetDBConnector interface {
	GetDBConn() (*sql.DB, error)
}

// Rows rows interface
type Rows interface {
	Columns() ([]string, error)
	ColumnTypes() ([]*sql.ColumnType, error)
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type ErrorTranslator interface {
	Translate(err error) error
}
