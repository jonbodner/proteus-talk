package main

import (
	"database/sql"
	"log"
	_ "github.com/lib/pq"
	"fmt"
)

type Person struct {
	Id   int `prof:"id"`
	Name string `prof:"name"`
	Age  int `prof:"age"`
}

func (p Person) String() string {
	return fmt.Sprintf("Id: %d\tName:%s\tAge:%d", p.Id, p.Name, p.Age)
}

type PersonDao struct {
	Create   func(e Executor, name string, age int) (int64, error) `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	Get      func(q Querier, id int) (*Person, error) `proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
	Update   func(e Executor, id int, name string, age int) (int64, error) `proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
	Delete   func(e Executor, id int) (int64, error) `proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
	GetAll   func(q Querier) ([]Person, error) `proq:"SELECT * FROM PERSON"`
	GetByAge func(q Querier, ages []int) ([]Person, error) `proq:"SELECT * from PERSON WHERE age in (:ages:)" prop:"ages"`
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
	fmt.Println("create:", count, err)

	count, err = personDao.Create(wrapper, "Bob", 50)
	fmt.Println("create #2:", count, err)

	count, err = personDao.Create(wrapper, "Julia", 32)
	fmt.Println("create #3:", count, err)

	person, err := personDao.Get(wrapper, 1)
	fmt.Println("get:", person, err)

	people, err := personDao.GetAll(wrapper)
	fmt.Println("get all:", people, err)

	people, err = personDao.GetByAge(wrapper, []int{20, 32})
	fmt.Println("get by age:", people, err)

	people, err = personDao.GetByAge(wrapper, []int{20, 32, 50})
	fmt.Println("get by age:", people, err)

	count, err = personDao.Update(wrapper, 1, "Freddie", 30)
	fmt.Println("update:", count, err)

	person, err = personDao.Get(wrapper, 1)
	fmt.Println("get #2:", person, err)

	count, err = personDao.Delete(wrapper, 1)
	fmt.Println("delete:", count, err)

	count, err = personDao.Delete(wrapper, 1)
	fmt.Println("delete #2:", count, err)

	person, err = personDao.Get(wrapper, 1)
	fmt.Println("get #3:", person, err)
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
