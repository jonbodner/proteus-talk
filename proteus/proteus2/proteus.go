package main

import (
	"reflect"
	"errors"
	"fmt"
)

func Build(dao interface{}) error {
	daoPointerType := reflect.TypeOf(dao)
	//must be a pointer to struct
	if daoPointerType.Kind() != reflect.Ptr {
		return errors.New("Not a pointer")
	}
	daoType := daoPointerType.Elem()
	//if not a struct, error out
	if daoType.Kind() != reflect.Struct {
		return errors.New("Not a pointer to struct")
	}
	daoPointerValue := reflect.ValueOf(dao)
	daoValue := reflect.Indirect(daoPointerValue)
	for i := 0; i < daoType.NumField(); i++ {
		curField := daoType.Field(i)
		query, ok := curField.Tag.Lookup("proq")
		if curField.Type.Kind() != reflect.Func || !ok {
			continue
		}
		funcType := curField.Type

		implementation, err := makeImplementation(funcType, query)
		if err != nil {
			continue
		}

		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
	}
	return nil
}

var exType = reflect.TypeOf((*Executor)(nil)).Elem()
var qType = reflect.TypeOf((*Querier)(nil)).Elem()

func makeImplementation(funcType reflect.Type, query string) (func([]reflect.Value) []reflect.Value, error) {
	if funcType.NumIn() == 0 {
		return nil, errors.New("need to supply an Executor or Querier parameter")
	}
	switch fType := funcType.In(0); {
	case fType.Implements(exType):
		return makeExecutorImplementation(funcType, query)
	case fType.Implements(qType):
		return makeQuerierImplementation(funcType, query)
	default:
		return nil, errors.New("first parameter must be of type api.Executor or api.Querier")
	}
}

func makeExecutorImplementation(funcType reflect.Type, query string) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		executor := args[0].Interface().(Executor)

		fmt.Println("I'm execing query",query)
		result, err := executor.Exec(query/*args are coming soon*/)
		fmt.Println("I got back results", result, "and error",err)

		return nil
	}, nil

}

func makeQuerierImplementation(funcType reflect.Type, query string) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		querier := args[0].Interface().(Querier)

		fmt.Println("I'm querying query",query)
		rows, err := querier.Query(query/*args are coming soon*/)
		fmt.Println("I got back rows", rows, "and error",err)

		return nil
	}, nil
}