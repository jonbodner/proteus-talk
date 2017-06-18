package main

import (
	"errors"
	"fmt"
	"reflect"
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

		fmt.Printf("Processing field %s with query %s\n", curField.Name, query)
		implementation, err := makeImplementation(funcType, query)
		if err != nil {
			continue
		}

		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
		fmt.Println()
	}
	return nil
}

func makeImplementation(funcType reflect.Type, query string) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		fmt.Printf("I'm a placeholder for query string %s\n", query)
		return nil
	}, nil
}
