package jsons

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// NewEmptyObj 初始化一个对象类型的 json 数据
func NewEmptyObj() *Item {
	return &Item{obj: make(map[string]*Item), jType: JsonTypeObj}
}

// NewEmptyArr 初始化一个数组类型的 json 数据
func NewEmptyArr() *Item {
	return &Item{arr: make([]*Item, 0), jType: JsonTypeArr}
}

// NewByObj 根据对象初始化 json 数据
func NewByObj(obj interface{}) *Item {
	if obj == nil {
		return NewByVal(nil)
	}

	if item, ok := obj.(*Item); ok {
		return item
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct && v.Kind() != reflect.Map {
		return NewByVal(obj)
	}

	item := NewEmptyObj()
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := v.Type().Field(i)
			item.obj[fieldType.Name] = NewByVal(fieldVal.Interface())
		}
	}

	if v.Kind() == reflect.Map {
		if v.Type().Key() != reflect.TypeOf("") {
			panic("不支持的 map 类型")
		}
		for _, key := range v.MapKeys() {
			item.obj[key.Interface().(string)] = NewByVal(v.MapIndex(key).Interface())
		}
	}

	return item
}

// NewByArr 根据数组初始化 json 数据
func NewByArr(arr interface{}) *Item {
	if arr == nil {
		return NewByVal(nil)
	}

	if item, ok := arr.(*Item); ok {
		return item
	}

	v := reflect.ValueOf(arr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		return NewByVal(arr)
	}

	item := NewEmptyArr()
	for i := 0; i < v.Len(); i++ {
		item.arr = append(item.arr, NewByVal(v.Index(i).Interface()))
	}
	return item
}

// NewByVal 根据指定普通值初始化 json 数据, 如果是数组或对象类型也会自动转化
func NewByVal(val interface{}) *Item {
	item := &Item{jType: JsonTypeVal}
	if val == nil {
		return item
	}

	switch newVal := val.(type) {
	case bool, int, float64, int64:
		item.val = newVal
		return item
	case string:
		// 将字符串中的 unicode 字符转换为 utf8
		if conv, err := strconv.Unquote(`"` + newVal + `"`); err == nil {
			item.val = conv
		} else if json, err := New(newVal); err == nil {
			return json
		} else {
			item.val = newVal
		}
		return item
	case *Item:
		return newVal
	default:
	}

	t := reflect.TypeOf(val)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Map {
		return NewByObj(val)
	}
	if t.Kind() == reflect.Array || t.Kind() == reflect.Slice {
		return NewByArr(val)
	}
	panic("无效的数据类型: " + t.Name())
}

// New 从 json 字符串中初始化成 item 对象
func New(rawJson string) (*Item, error) {
	if rawJson = strings.TrimSpace(rawJson); rawJson == "" {
		return NewByVal(rawJson), nil
	}

	if strings.HasPrefix(rawJson, `"`) && strings.HasSuffix(rawJson, `"`) {
		return NewByVal(rawJson[1 : len(rawJson)-1]), nil
	}

	if rawJson == "null" {
		return NewByVal(nil), nil
	}

	if strings.HasPrefix(rawJson, "{") {
		var data map[string]json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}
		item := NewEmptyObj()
		for key, value := range data {
			subI, err := New(string(value))
			if err != nil {
				return nil, err
			}
			item.Put(key, subI)
		}
		return item, nil
	}

	if strings.HasPrefix(rawJson, "[") {
		var data []json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}
		item := NewEmptyArr()
		for _, value := range data {
			subI, err := New(string(value))
			if err != nil {
				return nil, err
			}
			item.Append(subI)
		}
		return item, nil
	}

	// 尝试转换成基础类型
	var b bool
	if err := json.Unmarshal([]byte(rawJson), &b); err == nil {
		return NewByVal(b), nil
	}
	var i int
	if err := json.Unmarshal([]byte(rawJson), &i); err == nil {
		return NewByVal(i), nil
	}
	var f float64
	if err := json.Unmarshal([]byte(rawJson), &f); err == nil {
		return NewByVal(f), nil
	}

	return nil, fmt.Errorf("不支持的字符串: %s", rawJson)
}
