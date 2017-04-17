package tags

import (
	"reflect"
	"fmt"
)

func TagPrinter(s interface{}) {
	t := reflect.TypeOf(s)
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Printf("Field %s has %s for tag tag1 and %s for tag tag2\n", f.Name, f.Tag.Get("tag1"), f.Tag.Get("tag2"))
	}
}
