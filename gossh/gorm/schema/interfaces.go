package schema

import (
	"gossh/gorm/clause"
)

// ConstraintInterface database constraint interface
type ConstraintInterface interface {
	GetName() string
	Build() (sql string, vars []any)
}

// GormDataTypeInterface gorm data type interface
type GormDataTypeInterface interface {
	GormDataType() string
}

// FieldNewValuePool field new scan value pool
type FieldNewValuePool interface {
	Get() any
	Put(any)
}

// CreateClausesInterface create clauses interface
type CreateClausesInterface interface {
	CreateClauses(*Field) []clause.Interface
}

// QueryClausesInterface query clauses interface
type QueryClausesInterface interface {
	QueryClauses(*Field) []clause.Interface
}

// UpdateClausesInterface update clauses interface
type UpdateClausesInterface interface {
	UpdateClauses(*Field) []clause.Interface
}

// DeleteClausesInterface delete clauses interface
type DeleteClausesInterface interface {
	DeleteClauses(*Field) []clause.Interface
}
