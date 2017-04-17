package main

import (
	"reflect"
	"errors"
	"fmt"
	"bytes"
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
		nameOrderMap := buildNameOrderMap(paramOrder)

		implementation, err := makeImplementation(funcType, query, paramAdapter, nameOrderMap)
		if err != nil {
			return err
		}

		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
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

var errType = reflect.TypeOf((*error)(nil)).Elem()
var errZero = reflect.Zero(errType)

func makeExecutorImplementation(funcType reflect.Type, query string, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		executor := args[0].Interface().(Executor)

		queryArgs := buildQueryArgs(args, paramOrder)

		fmt.Println("I'm execing query", query, "with args", queryArgs)
		result, err := executor.Exec(query, queryArgs...)
		var count int64
		if err == nil {
			count, err = result.RowsAffected()
		}
		var errVal reflect.Value
		if err == nil {
			errVal = errZero
		} else {
			errVal = reflect.ValueOf(err).Convert(errType)
		}
		return []reflect.Value{reflect.ValueOf(count), errVal}

	}, nil
}

func makeQuerierImplementation(funcType reflect.Type, query string, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
	firstResult := funcType.Out(0)
	zeroVal := reflect.Zero(firstResult)
	returnType := firstResult.Elem()

	rowMapper := mapOneRow
	if firstResult.Kind() == reflect.Slice {
		rowMapper = func(rows Rows, mapper Mapper, zeroVal reflect.Value) (reflect.Value, error) {
			return mapAllRows(returnType, rows, mapper, zeroVal)
		}
	}

	mapper := buildMapper(returnType, zeroVal)

	return func(args []reflect.Value) []reflect.Value {
		querier := args[0].Interface().(Querier)

		queryArgs := buildQueryArgs(args, paramOrder)
		fmt.Println("I'm querying query", query, "with args", queryArgs)
		rows, err := querier.Query(query, queryArgs...)

		if err != nil {
			return []reflect.Value{zeroVal, reflect.ValueOf(err).Convert(errType)}
		}

		result, err := rowMapper(rows, mapper, zeroVal)
		rows.Close()

		if err != nil {
			return []reflect.Value{result, reflect.ValueOf(err).Convert(errType)}
		}

		return []reflect.Value{result, errZero}
	}, nil
}

func buildQueryArgs(funcArgs []reflect.Value, paramOrder []paramInfo) []interface{} {
	out := []interface{}{}
	for _, v := range paramOrder {
		out = append(out, funcArgs[v.posInParams].Interface())
	}
	return out
}

func mapOneRow(rows Rows, mapper Mapper, zeroVal reflect.Value) (reflect.Value, error) {
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return zeroVal, err
		}
		return zeroVal, nil
	}

	cols, err := rows.Columns()
	if err != nil {
		return zeroVal, err
	}

	vals := make([]interface{}, len(cols))
	for i := 0; i < len(vals); i++ {
		vals[i] = new(interface{})
	}

	err = rows.Scan(vals...)
	if err != nil {
		return zeroVal, err
	}

	return mapper(cols, vals)
}

func mapAllRows(returnType reflect.Type, rows Rows, mapper Mapper, zeroVal reflect.Value) (reflect.Value, error) {
	cols, err := rows.Columns()
	if err != nil {
		return zeroVal, err
	}

	outSlice := reflect.MakeSlice(reflect.SliceOf(returnType), 0, 0)

	for rows.Next() {
		if err := rows.Err(); err != nil {
			return zeroVal, err
		}

		vals := make([]interface{}, len(cols))
		for i := 0; i < len(vals); i++ {
			vals[i] = new(interface{})
		}

		err = rows.Scan(vals...)
		if err != nil {
			return zeroVal, err
		}
		curVal, err := mapper(cols, vals)
		if err != nil {
			return zeroVal, err
		}
		outSlice = reflect.Append(outSlice, curVal.Elem())
	}
	if err := rows.Err(); err != nil {
		return zeroVal, err
	}
	if outSlice.Len() == 0 {
		return zeroVal, nil
	}
	return outSlice, nil
}

type Mapper func(cols []string, vals []interface{}) (reflect.Value, error)

type fieldInfo struct {
	name      string
	fieldType reflect.Type
	pos       int
}

func buildMapper(returnType reflect.Type, zeroVal reflect.Value) Mapper {
	//build map of col names to field names (makes this 2N instead of N^2)
	colFieldMap := map[string]fieldInfo{}
	for i := 0; i < returnType.NumField(); i++ {
		sf := returnType.Field(i)
		tagVal := sf.Tag.Get("prof")
		colFieldMap[tagVal] = fieldInfo{
			name:      sf.Name,
			fieldType: sf.Type,
			pos:       i,
		}
	}

	return func(cols []string, vals []interface{}) (reflect.Value, error) {
		returnVal := reflect.New(returnType)
		err := populateReturnVal(returnVal, cols, vals, colFieldMap)
		if err != nil {
			return zeroVal, err
		}
		return returnVal, err
	}
}

func populateReturnVal(returnVal reflect.Value, cols []string, vals []interface{}, colFieldMap map[string]fieldInfo) error {
	val := returnVal.Elem()
	for k, v := range cols {
		if sf, ok := colFieldMap[v]; ok {
			curVal := vals[k]
			rv := reflect.ValueOf(curVal)
			if rv.Elem().Elem().Type().ConvertibleTo(sf.fieldType) {
				val.Field(sf.pos).Set(rv.Elem().Elem().Convert(sf.fieldType))
			} else {
				return fmt.Errorf("Unable to assign value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.name, sf.fieldType)
			}
		}
	}
	return nil
}
