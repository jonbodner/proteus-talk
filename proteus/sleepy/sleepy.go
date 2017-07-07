package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"text/template"
)

func Build(dao interface{}, buildData BuildData) error {
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
		query, ok := curField.Tag.Lookup("sleq")
		if curField.Type.Kind() != reflect.Func || !ok {
			continue
		}
		funcType := curField.Type

		paramOrder := curField.Tag.Get("slep")
		nameOrderMap := buildNameOrderMap(paramOrder)

		responseType := curField.Tag.Get("sler")
		if responseType == "" {
			responseType = "Body"
		}

		implementation, err := makeImplementation(funcType, query, nameOrderMap, responseType, buildData)
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
		out[v] = k
	}
	return out
}

func buildMap(in []reflect.Value, nameOrderMap map[string]int) map[string]interface{} {
	m := map[string]interface{}{}
	for k, v := range nameOrderMap {
		m[k] = in[v].Interface()
	}
	return m
}

var errType = reflect.TypeOf((*error)(nil)).Elem()
var errZero = reflect.Zero(errType)

var client = &http.Client{}

func makeImplementation(funcType reflect.Type, query string, nameOrderMap map[string]int, responseType string, buildData BuildData) (func([]reflect.Value) []reflect.Value, error) {
	//split the query on space to get the method
	parts := strings.Split(query, " ")
	if len(parts) < 2 {
		return nil, fmt.Errorf("Unexpected query structure. Expected METHOD PATH BODY, got: %s", query)
	}
	returnType := funcType.Out(0)
	returnTypeZero := reflect.Zero(returnType)
	prefix := buildData.ToPrefix()
	queryHolder, err := buildFixedQuery(parts[1])
	if err != nil {
		return nil, err
	}

	return func(in []reflect.Value) []reflect.Value {
		vals := buildMap(in, nameOrderMap)
		path, err := queryHolder.finalize(vals)
		if err != nil {
			return []reflect.Value{returnTypeZero, reflect.ValueOf(err).Convert(errType)}
		}

		var outBody io.Reader

		if len(parts) == 3 {
			val := in[nameOrderMap[parts[2]]].Interface()
			b, err := json.Marshal(val)
			if err != nil {
				return []reflect.Value{returnTypeZero, reflect.ValueOf(err).Convert(errType)}
			}
			outBody = bytes.NewReader(b)
		}

		url := fmt.Sprintf("%s%s", prefix, path)

		req, err := http.NewRequest(parts[0], url, outBody)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			return []reflect.Value{returnTypeZero, reflect.ValueOf(err).Convert(errType)}
		}
		if responseType == STATUS_CODE {
			return []reflect.Value{reflect.ValueOf(resp.StatusCode), errZero}
		}
		if responseType == BODY {
			rt := returnType
			if rt.Kind() == reflect.Ptr {
				rt = rt.Elem()
			}
			newInstance := reflect.New(rt)
			i := map[string]interface{}{}
			unmarshaler := json.NewDecoder(resp.Body)
			err = unmarshaler.Decode(&i)
			if err != nil {
				return []reflect.Value{returnTypeZero, reflect.ValueOf(err).Convert(errType)}
			}
			populate(newInstance.Elem(), i)
			return []reflect.Value{newInstance, errZero}
		}
		result := resp.Header.Get(strings.Split(responseType, ":")[1])
		return []reflect.Value{reflect.ValueOf(result), errZero}
	}, nil
}

func populate(r reflect.Value, m map[string]interface{}) {
	rt := r.Type()
	for i := 0; i < r.NumField(); i++ {
		r.Field(i).Set(reflect.ValueOf(m[rt.Field(i).Tag.Get("json")]).Convert(rt.Field(i).Type))
	}
}

type paramInfo struct {
	name        string
	posInParams int
}

func buildFixedQuery(query string) (queryHolder, error) {
	var out bytes.Buffer

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
		case '{':
			inParam = !inParam
		case '}':
			name := curName.String()
			out.WriteString(fmt.Sprintf(sliceTemplate, name))
			curName.Reset()
		default:
			if !inParam {
				out.WriteRune(v)
			} else {
				curName.WriteRune(v)
			}
		}
	}

	queryString := out.String()

	return templateQueryHolder(queryString), nil
}

// template slice support
type queryHolder interface {
	finalize(vals map[string]interface{}) (string, error)
}

type templateQueryHolder string

func (tq templateQueryHolder) finalize(vals map[string]interface{}) (string, error) {
	return doFinalize(string(tq), vals)
}

func doFinalize(queryString string, vals map[string]interface{}) (string, error) {
	temp, err := template.New("query").Parse(queryString)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	err = temp.Execute(&b, vals)
	if err != nil {
		return "", err
	}
	return b.String(), err
}

const (
	sliceTemplate = `{{.%s}}`
)
