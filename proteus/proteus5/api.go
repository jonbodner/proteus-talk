package main

import "database/sql"

type Rows interface {
	Next() bool

	Err() error

	Columns() ([]string, error)

	Scan(dest ...interface{}) error

	Close() error
}

// Executor runs queries that modify the data store.
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Querier runs queries that return Rows from the data store
type Querier interface {
	Query(query string, args ...interface{}) (Rows, error)
}

type Wrapper interface {
	Executor
	Querier
}

type ParamAdapter func(pos int) string
