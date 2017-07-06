package main

import "database/sql"

func Adapt(sqle Sql) Wrapper {
	return sqlWrapper{sqle}
}

type sqlWrapper struct {
	Sql
}

func (w sqlWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return w.Sql.Exec(query, args...)
}

func (w sqlWrapper) Query(query string, args ...interface{}) (Rows, error) {
	return w.Sql.Query(query, args...)
}

// sqlExecutor matches the interface provided by several types in the standard go sql package.
type Sql interface {
	Exec(query string, args ...interface{}) (sql.Result, error)

	Query(query string, args ...interface{}) (*sql.Rows, error)
}
