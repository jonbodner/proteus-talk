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
	Create func(e Executor, name string, age int) (int64, error)         `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	Get    func(q Querier, id int)                                       `proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
	Update func(e Executor, id int, name string, age int) (int64, error) `proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
	Delete func(e Executor, id int) (int64, error)                       `proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
}

var personDao PersonDao

func init() {
	fmt.Println("Setting up the DAO")
	err := Build(&personDao, Postgres)
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
		count, err := personDao.Create(wrapper, "Fred", 20)
		fmt.Println("create: number of rows", count, "with error", err)
	})

	doStep("Get", func() {
		personDao.Get(wrapper, 1)
	})

	doStep("Update", func() {
		count, err := personDao.Update(wrapper, 1, "Freddie", 30)
		fmt.Println("update: number of rows", count, "with error", err)
	})

	doStep("Delete", func() {
		count, err := personDao.Delete(wrapper, 1)
		fmt.Println("delete: number of rows", count, "with error", err)
	})

	doStep("Delete #2", func() {
		count, err := personDao.Delete(wrapper, 1)
		fmt.Println("delete #2: number of rows", count, "with error", err)
	})
}

func main() {
	db := setupDbPostgres()
	wrapper := Adapt(db)
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
