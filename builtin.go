package cog

import (
	"fmt"
	"reflect"
	"unsafe"
)

func If[T any](condition bool, consequence T, alternative ...T) T {
	if condition {
		return consequence
	}

	if len(alternative) == 0 {
		return *new(T)
	}

	if len(alternative) > 1 {
		panic("@if: wrong number of arguments")
	}

	return alternative[0]
}

var reflectStringer = reflect.TypeFor[fmt.Stringer]()

func Print(msg any) {
	val := reflect.ValueOf(msg)

	switch val.Kind() {
	case reflect.String:
		fmt.Println(val.String())
	case reflect.Slice:
		if val.Index(0).Kind() == reflect.Uint8 {
			b := val.Bytes()
			fmt.Println(unsafe.String(&b[0], len(b)))
		} else {
			fmt.Printf("%v\n", msg)
		}
	case reflect.Struct:
		fmt.Printf("%+v\n", msg)
	case reflect.Int32:
		if r, ok := msg.(rune); ok {
			fmt.Printf("%c\n", r)
		}
	default:
		if val.Type().Implements(reflectStringer) {
			fmt.Println(msg.(fmt.Stringer).String())
		} else {
			fmt.Printf("%v\n", msg)
		}
	}
}
