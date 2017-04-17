package main

type PersonDao struct {
	Create func(/*Parameters*/) `proq:"CREATE SQL QUERY GOES HERE"`
	Get    func(/*Parameters*/) `proq:"READ SQL QUERY GOES HERE"`
	Update func(/*Parameters*/) `proq:"UPDATE SQL QUERY GOES HERE"`
	Delete func(/*Parameters*/) `proq:"DELETE SQL QUERY GOES HERE"`
}

var personDao PersonDao

func init() {
	err := Build(&personDao)
	if err != nil {
		panic(err)
	}
}

func DoPersonStuff() {
	personDao.Create()
	personDao.Get()
	personDao.Update()
	personDao.Delete()
}

func main() {
	DoPersonStuff()
}
