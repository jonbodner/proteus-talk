package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

type PersonDao struct {
	Create func(Executor /*Parameters*/) `proq:"CREATE SQL QUERY GOES HERE"`
	Get    func(Querier /*Parameters*/)  `proq:"READ SQL QUERY GOES HERE"`
	Update func(Executor /*Parameters*/) `proq:"UPDATE SQL QUERY GOES HERE"`
	Delete func(Executor /*Parameters*/) `proq:"DELETE SQL QUERY GOES HERE"`
}

var personDao PersonDao

func init() {
	fmt.Println("Setting up the DAO")
	err := Build(&personDao)
	fmt.Println("DAO created")
	if err != nil {
		panic(err)
	}
}

func doStep(label string, f func()) {
	fmt.Println()
	fmt.Println(label)
	f()
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

func DoPersonStuff(wrapper Wrapper) {
	doStep("Create", func() {
		personDao.Create(wrapper)
	})

	doStep("Get", func() {
		personDao.Get(wrapper)
	})

	doStep("Update", func() {
		personDao.Update(wrapper)
	})

	doStep("Delete", func() {
		personDao.Delete(wrapper)
	})
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
	create table person (id serial primary key, name text, age int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}
	return db
}
