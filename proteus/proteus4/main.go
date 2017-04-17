package main

import (
	"database/sql"
	"log"
	_ "github.com/lib/pq"
	"fmt"
)

type PersonDao struct {
	Create func(e Executor, name string, age int) (int64, error) `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	Get    func(q Querier, id int) `proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
	Update func(e Executor, id int, name string, age int) (int64, error) `proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
	Delete func(e Executor, id int) (int64, error) `proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
}

var personDao PersonDao

func init() {
	err := Build(&personDao, Postgres)
	if err != nil {
		panic(err)
	}
}

func DoPersonStuff(wrapper Wrapper) {
	count, err := personDao.Create(wrapper, "Fred", 20)
	fmt.Println("create",count,err)
	personDao.Get(wrapper, 1)
	count, err = personDao.Update(wrapper, 1, "Freddie", 30)
	fmt.Println("update",count,err)
	personDao.Get(wrapper, 1)
	count, err = personDao.Delete(wrapper, 1)
	fmt.Println("delete",count,err)
	count, err = personDao.Delete(wrapper, 1)
	fmt.Println("delete",count,err)
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
