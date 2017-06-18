package main

import (
	"bufio"
	"fmt"
	"os"
)

type PersonDao struct {
	Create func( /*Parameters*/ ) `proq:"CREATE SQL QUERY GOES HERE"`
	Get    func( /*Parameters*/ ) `proq:"READ SQL QUERY GOES HERE"`
	Update func( /*Parameters*/ ) `proq:"UPDATE SQL QUERY GOES HERE"`
	Delete func( /*Parameters*/ ) `proq:"DELETE SQL QUERY GOES HERE"`
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

func DoPersonStuff() {
	doStep("Create", func() {
		personDao.Create()
	})

	doStep("Get", func() {
		personDao.Get()
	})

	doStep("Update", func() {
		personDao.Update()
	})

	doStep("Delete", func() {
		personDao.Delete()
	})
}

func main() {
	DoPersonStuff()
}
