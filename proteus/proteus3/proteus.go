package main

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func Build(dao interface{}, paramAdapter ParamAdapter) error {
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

		paramOrder := curField.Tag.Get("prop")

		fmt.Printf("Processing field %s with query %s and paramOrder %s\n", curField.Name, query, paramOrder)
		nameOrderMap := buildNameOrderMap(paramOrder)

		implementation, err := makeImplementation(funcType, query, paramAdapter, nameOrderMap)
		if err != nil {
			return err
		}

		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
		fmt.Println()
	}
	return nil
}

func buildNameOrderMap(paramOrder string) map[string]int {
	out := map[string]int{}
	params := strings.Split(paramOrder, ",")
	for k, v := range params {
		out[v] = k + 1
	}
	return out
}

var exType = reflect.TypeOf((*Executor)(nil)).Elem()
var qType = reflect.TypeOf((*Querier)(nil)).Elem()

func makeImplementation(funcType reflect.Type, query string, paramAdapter ParamAdapter, nameOrderMap map[string]int) (func([]reflect.Value) []reflect.Value, error) {
	if funcType.NumIn() == 0 {
		return nil, errors.New("need to supply an Executor or Querier parameter")
	}
	switch fType := funcType.In(0); {
	case fType.Implements(exType):
		fixedQuery, paramOrder := buildFixedQueryAndParamOrder(query, nameOrderMap, paramAdapter)
		return makeExecutorImplementation(funcType, fixedQuery, paramOrder)
	case fType.Implements(qType):
		fixedQuery, paramOrder := buildFixedQueryAndParamOrder(query, nameOrderMap, paramAdapter)
		return makeQuerierImplementation(funcType, fixedQuery, paramOrder)
	default:
		return nil, errors.New("first parameter must be of type api.Executor or api.Querier")
	}
}

type paramInfo struct {
	name        string
	posInParams int
}

func buildFixedQueryAndParamOrder(query string, nameOrderMap map[string]int, paramAdapter ParamAdapter) (string, []paramInfo) {
	pos := 1
	var out bytes.Buffer
	var paramOrder []paramInfo

	isEscaped := false
	inParam := false
	var curName bytes.Buffer
	for _, v := range query {
		if isEscaped {
			out.WriteRune(v)
			isEscaped = false
			continue
		}
		switch v {
		case '\\':
			isEscaped = true
		case ':':
			if inParam {
				out.WriteString(paramAdapter(pos))
				name := curName.String()
				paramOrder = append(paramOrder, paramInfo{name: name, posInParams: nameOrderMap[name]})
				pos++
				curName.Reset()
			}
			inParam = !inParam
		default:
			if !inParam {
				out.WriteRune(v)
			} else {
				curName.WriteRune(v)
			}
		}
	}
	return out.String(), paramOrder
}

func makeExecutorImplementation(funcType reflect.Type, query string, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		executor := args[0].Interface().(Executor)

		queryArgs := buildQueryArgs(args, paramOrder)

		fmt.Println("I'm execing query", query, "with args", queryArgs)
		result, err := executor.Exec(query, queryArgs...)
		fmt.Println("I got back results", result, "and error", err)

		return nil
	}, nil
}

func makeQuerierImplementation(funcType reflect.Type, query string, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		querier := args[0].Interface().(Querier)

		queryArgs := buildQueryArgs(args, paramOrder)
		fmt.Println("I'm querying query", query, "with args", queryArgs)
		rows, err := querier.Query(query, queryArgs...)
		fmt.Println("I got back rows", rows, "and error", err)

		return nil
	}, nil
}

func buildQueryArgs(funcArgs []reflect.Value, paramOrder []paramInfo) []interface{} {
	out := []interface{}{}
	for _, v := range paramOrder {
		out = append(out, funcArgs[v.posInParams].Interface())
	}
	return out
}
