package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Person struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (p Person) String() string {
	return fmt.Sprintf("{Id: %d, Name:%s, Age:%d}", p.Id, p.Name, p.Age)
}

type PersonClient struct {
	Create func(body Person) (string, error)      `sleq:"POST /person body"   slep:"body"    sler:"Header:X-ID"`
	Get    func(id int) (*Person, error)          `sleq:"GET /person/{id}"    slep:"id"`
	Update func(body Person, id int) (int, error) `sleq:"PUT /person/{id} body"    slep:"body,id" sler:"Status"`
	Delete func(id int) (int, error)              `sleq:"DELETE /person/{id}" slep:"id"      sler:"Status"`
}

var personClient PersonClient

func init() {
	err := Build(&personClient, BuildData{
		Host: "localhost",
		Port: 8080,
	})
	if err != nil {
		panic(err)
	}
}

func DoPersonStuff() {
	location, err := personClient.Create(Person{
		Name: "Fred",
		Age:  20,
	})
	fmt.Println("create:", location, err)
	id, _ := strconv.Atoi(location)

	person, err := personClient.Get(id)
	fmt.Println("get:", person, err)

	count, err := personClient.Update(Person{
		Name: "Freddie",
		Age:  30,
	}, id)
	fmt.Println("update:", count, err)

	person, err = personClient.Get(id)
	fmt.Println("get #2:", person, err)

	count, err = personClient.Delete(id)
	fmt.Println("delete:", count, err)
}

func main() {
	LaunchServer()
	DoPersonStuff()
}

func LaunchServer() {
	r := mux.NewRouter()

	s := &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	curId := 1
	m := map[int]Person{}

	r.Methods("POST").Path("/person").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var p Person
		unmarshaler := json.NewDecoder(req.Body)
		unmarshaler.Decode(&p)
		p.Id = curId

		m[curId] = p
		w.Header().Add("X-ID", strconv.Itoa(curId))
		curId++
	})

	r.Methods("GET").Path("/person/{id}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		id, _ := strconv.Atoi(vars["id"])
		p := m[id]
		b, _ := json.Marshal(p)
		w.Write(b)
	})

	r.Methods("PUT").Path("/person/{id}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var p Person
		unmarshaler := json.NewDecoder(req.Body)
		unmarshaler.Decode(&p)

		vars := mux.Vars(req)
		id, _ := strconv.Atoi(vars["id"])

		p.Id = id
		m[id] = p
	})

	r.Methods("DELETE").Path("/person/{id}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		id, _ := strconv.Atoi(vars["id"])
		delete(m, id)
		w.WriteHeader(204)
	})

	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}
	go s.Serve(l)
}
