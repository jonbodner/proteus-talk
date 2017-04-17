package main

import (
	"database/sql"
	"log"
	_ "github.com/lib/pq"
)

type PersonDao struct {
	Create func(Executor/*Parameters*/) `proq:"CREATE SQL QUERY GOES HERE"`
	Get    func(Querier/*Parameters*/) `proq:"READ SQL QUERY GOES HERE"`
	Update func(Executor/*Parameters*/) `proq:"UPDATE SQL QUERY GOES HERE"`
	Delete func(Executor/*Parameters*/) `proq:"DELETE SQL QUERY GOES HERE"`
}

var personDao PersonDao

func init() {
	err := Build(&personDao)
	if err != nil {
		panic(err)
	}
}

func DoPersonStuff(wrapper Wrapper) {
	personDao.Create(wrapper)
	personDao.Get(wrapper)
	personDao.Update(wrapper)
	personDao.Delete(wrapper)
}

func main() {
	db := setupDbPostgres()
	wrapper := Wrap(db)
	DoPersonStuff(wrapper)
}

func setupDbPostgres() *sql.DB {
	db, err := sql.Open("postgres", "postgres://jon:jon@localhost/jon?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	drop table if exists person;
	create table person (id integer not null primary key, name text, age int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}
	return db
}
