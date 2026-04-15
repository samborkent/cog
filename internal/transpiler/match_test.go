package transpiler_test

import "testing"

func TestConvertMatchGeneric(t *testing.T) {
	t.Parallel()

	t.Run("basic_type_switch", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match x {
	case int32:
		@print(x)
	case utf8:
		@print(x)
	}
}
main : proc() = {}`)

		mustContain(t, got, "switch")
		mustContain(t, got, "any(x)")
		mustContain(t, got, ".(type)")
		mustContain(t, got, "case int32")
		mustContain(t, got, "case string")
	})

	t.Run("with_binding", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match val := x {
	case int32:
		@print(val)
	case utf8:
		@print(val)
	}
}
main : proc() = {}`)

		mustContain(t, got, "val :=")
		mustContain(t, got, "any(x)")
		mustContain(t, got, ".(type)")
	})

	t.Run("with_default", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case int32:
		@print(x)
	default:
		@print(x)
	}
}
main : proc() = {}`)

		mustContain(t, got, "switch")
		mustContain(t, got, "case int32")
		mustContain(t, got, "default:")
	})

	t.Run("any_constraint", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case int64:
		@print(x)
	case utf8:
		@print(x)
	}
}
main : proc() = {}`)

		mustContain(t, got, "switch")
		mustContain(t, got, "any(x)")
		mustContain(t, got, "case int64")
		mustContain(t, got, "case string")
	})

	t.Run("binding_with_default", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ any>(x : T) = {
	match val := x {
	case int64:
		@print(val)
	default:
		@print(val)
	}
}
main : proc() = {}`)

		mustContain(t, got, "val :=")
		mustContain(t, got, "default:")
	})

	t.Run("multiple_cases", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ int8 | int16 | int32>(x : T) = {
	match x {
	case int8:
		@print(x)
	case int16:
		@print(x)
	case int32:
		@print(x)
	}
}
main : proc() = {}`)

		mustContain(t, got, "case int8")
		mustContain(t, got, "case int16")
		mustContain(t, got, "case int32")
	})

	t.Run("no_binding_type_assert", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match x {
	case int32:
		@print(x)
	}
}
main : proc() = {}`)

		mustContain(t, got, "switch")
		mustContain(t, got, ".(type)")
	})
}
