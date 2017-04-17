package generate

import (
	"reflect"
)

type Calculator func(a, b int) int

func MemoizeCalculator(c Calculator) Calculator {
	t := reflect.TypeOf(c)
	type vals struct {
		a,b int
	}
	cache := map[vals]int{}
	v := reflect.MakeFunc(t, func(args []reflect.Value) []reflect.Value {
		a := args[0].Interface().(int)
		b := args[1].Interface().(int)
		v := vals{a:a, b:b}
		result, ok := cache[v]
		if !ok {
			result = c(a,b)
			cache[v] = result
		}
		return []reflect.Value{reflect.ValueOf(result)}
	})
	return v.Interface().(Calculator)
}
