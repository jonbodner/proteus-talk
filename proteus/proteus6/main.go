package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

type Person struct {
	Id   int    `prof:"id"`
	Name string `prof:"name"`
	Age  int    `prof:"age"`
}

func (p Person) String() string {
	return fmt.Sprintf("{Id: %d, Name:%s, Age:%d}", p.Id, p.Name, p.Age)
}

type PersonDao struct {
	Create func(e Executor, name string, age int) (int64, error)         `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	Get    func(q Querier, id int) (*Person, error)                      `proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
	Update func(e Executor, id int, name string, age int) (int64, error) `proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
	Delete func(e Executor, id int) (int64, error)                       `proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
	GetAll func(q Querier) ([]Person, error)                             `proq:"SELECT * FROM PERSON"`
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
	doStep("Create #1", func() {
		count, err := personDao.Create(wrapper, "Fred", 20)
		fmt.Println("create: number of rows", count, "with error", err)
	})

	doStep("Create #2", func() {
		count, err := personDao.Create(wrapper, "Bob", 50)
		fmt.Println("create #2:  number of rows", count, "with error", err)
	})

	doStep("Create #3", func() {
		count, err := personDao.Create(wrapper, "Julia", 32)
		fmt.Println("create #3:  number of rows", count, "with error", err)
	})

	doStep("Get all", func() {
		people, err := personDao.GetAll(wrapper)
		fmt.Println("get all: result", people, "with error", err)
	})

	doStep("Update", func() {
		count, err := personDao.Update(wrapper, 1, "Freddie", 30)
		fmt.Println("update: number of rows", count, "with error", err)
	})

	doStep("Get all #2", func() {
		people, err := personDao.GetAll(wrapper)
		fmt.Println("get all #2: result", people, "with error", err)
	})

	doStep("Delete", func() {
		count, err := personDao.Delete(wrapper, 1)
		fmt.Println("delete: number of rows", count, "with error", err)
	})

	doStep("Get all #3", func() {
		people, err := personDao.GetAll(wrapper)
		fmt.Println("get all #3: result", people, "with error", err)
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
