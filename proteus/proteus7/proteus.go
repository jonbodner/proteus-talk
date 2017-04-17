package main

import (
	"reflect"
	"errors"
	"fmt"
	"bytes"
	"strings"
	"text/template"
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
		fixedQuery, paramOrder, err := buildFixedQueryAndParamOrder(query, nameOrderMap, funcType, paramAdapter)
		if err != nil {
			return nil, err
		}
		return makeExecutorImplementation(funcType, fixedQuery, paramOrder)
	case fType.Implements(qType):
		fixedQuery, paramOrder, err := buildFixedQueryAndParamOrder(query, nameOrderMap, funcType, paramAdapter)
		if err != nil {
			return nil, err
		}
		return makeQuerierImplementation(funcType, fixedQuery, paramOrder)
	default:
		return nil, errors.New("first parameter must be of type api.Executor or api.Querier")
	}
}

type paramInfo struct {
	name        string
	posInParams int
	isSlice     bool
}

func buildFixedQueryAndParamOrder(query string, nameOrderMap map[string]int, funcType reflect.Type, pa ParamAdapter) (queryHolder, []paramInfo, error) {
	var out bytes.Buffer
	var paramOrder []paramInfo

	isEscaped := false
	inParam := false
	var curName bytes.Buffer
	hasSlice := false
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
				name := curName.String()
				out.WriteString(fmt.Sprintf(sliceTemplate, name))

				//let's see if this is a slice or not
				paramPos := nameOrderMap[name]
				isSlice := false
				if funcType.In(paramPos).Kind() == reflect.Slice {
					isSlice = true
					hasSlice = true
				}
				paramOrder = append(paramOrder, paramInfo{name: name, posInParams: paramPos, isSlice: isSlice})
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

	queryString := out.String()

	if !hasSlice {
		//no slices, so last param is never going to be referenced in doFinalize
		queryString, err := doFinalize(queryString, paramOrder, pa, nil)
		if err != nil {
			return nil, nil, err
		}
		return simpleQueryHolder(queryString), paramOrder, nil
	}
	return templateQueryHolder{queryString:queryString, pa: pa, paramOrder: paramOrder}, paramOrder, nil
}

var errType = reflect.TypeOf((*error)(nil)).Elem()
var errZero = reflect.Zero(errType)

func makeExecutorImplementation(funcType reflect.Type, query queryHolder, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
	return func(args []reflect.Value) []reflect.Value {
		executor := args[0].Interface().(Executor)

		finalQuery, err := query.finalize(args)
		if err != nil {
			var count int64
			return []reflect.Value{reflect.ValueOf(count), reflect.ValueOf(err).Convert(errType)}
		}

		queryArgs := buildQueryArgs(args, paramOrder)

		fmt.Println("I'm execing query", finalQuery, "with args", queryArgs)
		result, err := executor.Exec(finalQuery, queryArgs...)
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

func makeQuerierImplementation(funcType reflect.Type, query queryHolder, paramOrder []paramInfo) (func([]reflect.Value) []reflect.Value, error) {
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

		finalQuery, err := query.finalize(args)
		if err != nil {
			return []reflect.Value{zeroVal, reflect.ValueOf(err).Convert(errType)}
		}

		queryArgs := buildQueryArgs(args, paramOrder)
		fmt.Println("I'm querying query", finalQuery, "with args", queryArgs)
		rows, err := querier.Query(finalQuery, queryArgs...)

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
		if v.isSlice {
			curSlice := funcArgs[v.posInParams]
			for i := 0; i < curSlice.Len(); i++ {
				out = append(out, curSlice.Index(i).Interface())
			}
		} else {
			out = append(out, funcArgs[v.posInParams].Interface())
		}
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


// template slice support
type queryHolder interface {
	finalize(args []reflect.Value) (string, error)
}

type simpleQueryHolder string

func (sq simpleQueryHolder) finalize(args []reflect.Value) (string, error) {
	return string(sq), nil
}

type templateQueryHolder struct {
	queryString string
	pa          ParamAdapter
	paramOrder  []paramInfo
}

func (tq templateQueryHolder) finalize(args []reflect.Value) (string, error) {
	return doFinalize(tq.queryString, tq.paramOrder, tq.pa, args)
}

func doFinalize(queryString string, paramOrder []paramInfo, pa ParamAdapter, args []reflect.Value) (string, error) {
	temp, err := template.New("query").Funcs(template.FuncMap{"join": joinFactory(1, pa)}).Parse(queryString)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	sliceMap := map[string]interface{}{}
	for _, v := range paramOrder {
		if v.isSlice {
			sliceMap[v.name] = args[v.posInParams].Len()
		} else {
			sliceMap[v.name] = 1
		}
	}
	if err == nil {
		fmt.Println("Finalizing query", queryString, "with values", sliceMap)
		err = temp.Execute(&b, sliceMap)
	}
	if err != nil {
		return "", err
	}
	return b.String(), err
}

const (
	sliceTemplate = `{{.%s | join}}`
)

func joinFactory(startPos int, paramAdapter ParamAdapter) func(int) string {
	return func(total int) string {
		var b bytes.Buffer
		for i := 0; i < total; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(paramAdapter(startPos + i))
		}
		startPos += total
		return b.String()
	}
}
