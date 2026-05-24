package builtin

import (
	"fmt"
	"reflect"
	"unsafe"
)

var reflectStringer = reflect.TypeFor[fmt.Stringer]()

func Print(msg any) {
	val := reflect.ValueOf(msg)

	switch val.Kind() {
	case reflect.String:
		fmt.Println(val.String())
	case reflect.Slice:
		switch val.Index(0).Kind() {
		case reflect.Uint8:
			b := val.Bytes()
			fmt.Println(unsafe.String(&b[0], len(b)))
		case reflect.Int32:
			runes, ok := val.Interface().([]rune)
			if ok {
				fmt.Println(string(runes))
				return
			}

			fallthrough
		default:
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
