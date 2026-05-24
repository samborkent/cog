package builtin

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

var (
	reflectStringer = reflect.TypeFor[fmt.Stringer]()
	emptyStructType = reflect.TypeOf(struct{}{})
)

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
		t := val.Type()

		if t.Implements(reflectStringer) {
			fmt.Println(msg.(fmt.Stringer).String())
		} else if t.Kind() == reflect.Map &&
			t.Key().Comparable() &&
			t.Elem() == emptyStructType {
			printSet(val)
		} else {
			fmt.Printf("%v\n", msg)
		}
	}
}

func printSet(v reflect.Value) {
	var out strings.Builder

	_, _ = out.WriteString("set[")

	keys := v.MapKeys()

	for i, k := range keys {
		fmt.Fprintf(&out, "%s", k)

		if i < len(keys)-1 {
			_ = out.WriteByte(' ')
		}
	}

	_ = out.WriteByte(']')

	fmt.Println(out.String())
}
