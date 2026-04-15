package types

import (
	"strings"
	"testing"
)

func TestInterface(t *testing.T) {
	t.Parallel()

	proc := &Procedure{
		Function:   true,
		ReturnType: Basics[UTF8],
	}

	iface := &Interface{
		Methods: []*Method{
			{Name: "String", Procedure: proc},
		},
	}

	t.Run("kind", func(t *testing.T) {
		t.Parallel()

		if iface.Kind() != InterfaceKind {
			t.Errorf("Kind() = %v, want InterfaceKind", iface.Kind())
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := iface.String()
		if s == "" {
			t.Error("String() returned empty")
		}

		if !strings.Contains(s, "interface") {
			t.Errorf("String() = %q, missing 'interface'", s)
		}

		if !strings.Contains(s, "String") {
			t.Errorf("String() = %q, missing method name", s)
		}
	})

	t.Run("underlying", func(t *testing.T) {
		t.Parallel()

		if iface.Underlying() != iface {
			t.Error("Underlying() should return self")
		}
	})

	t.Run("empty_interface", func(t *testing.T) {
		t.Parallel()

		empty := &Interface{}
		if empty.Kind() != InterfaceKind {
			t.Errorf("empty interface Kind() = %v", empty.Kind())
		}

		s := empty.String()
		if !strings.Contains(s, "interface") {
			t.Errorf("empty String() = %q", s)
		}
	})

	t.Run("multi_method_string", func(t *testing.T) {
		t.Parallel()

		multi := &Interface{
			Methods: []*Method{
				{Name: "Read", Procedure: &Procedure{Function: true, ReturnType: Basics[Int64]}},
				{Name: "Write", Procedure: &Procedure{Function: false}},
			},
		}

		s := multi.String()
		if !strings.Contains(s, "Read") || !strings.Contains(s, "Write") {
			t.Errorf("multi-method String() = %q", s)
		}
	})
}

func TestImplements(t *testing.T) {
	t.Parallel()

	stringProc := &Procedure{
		Function:   true,
		ReturnType: Basics[UTF8],
	}
	stringer := &Interface{
		Methods: []*Method{
			{Name: "String", Procedure: stringProc},
		},
	}

	t.Run("struct_with_method", func(t *testing.T) {
		t.Parallel()

		s := &Struct{
			Methods: []*Method{
				{Name: "String", Procedure: stringProc},
			},
		}

		if !Implements(s, stringer) {
			t.Error("struct with String method should implement Stringer")
		}
	})

	t.Run("struct_missing_method", func(t *testing.T) {
		t.Parallel()

		s := &Struct{}
		if Implements(s, stringer) {
			t.Error("struct without methods should not implement Stringer")
		}
	})

	t.Run("struct_wrong_signature", func(t *testing.T) {
		t.Parallel()

		s := &Struct{
			Methods: []*Method{
				{Name: "String", Procedure: &Procedure{
					Function:   true,
					ReturnType: Basics[Int64], // wrong return type
				}},
			},
		}

		if Implements(s, stringer) {
			t.Error("struct with wrong return type should not implement Stringer")
		}
	})

	t.Run("alias_wrapping_struct", func(t *testing.T) {
		t.Parallel()

		s := &Struct{
			Methods: []*Method{
				{Name: "String", Procedure: stringProc},
			},
		}
		alias := &Alias{Name: "Foo", Derived: s}

		if !Implements(alias, stringer) {
			t.Error("alias wrapping struct with method should implement Stringer")
		}
	})

	t.Run("non_struct_type", func(t *testing.T) {
		t.Parallel()

		if Implements(Basics[Int64], stringer) {
			t.Error("basic type should not implement interface")
		}
	})

	t.Run("empty_interface", func(t *testing.T) {
		t.Parallel()

		empty := &Interface{}
		s := &Struct{}

		if !Implements(s, empty) {
			t.Error("any struct should implement empty interface")
		}
	})

	t.Run("multi_method_interface", func(t *testing.T) {
		t.Parallel()

		readWrite := &Interface{
			Methods: []*Method{
				{Name: "Read", Procedure: &Procedure{Function: true, ReturnType: Basics[Int64]}},
				{Name: "Write", Procedure: &Procedure{Function: false}},
			},
		}

		full := &Struct{
			Methods: []*Method{
				{Name: "Read", Procedure: &Procedure{Function: true, ReturnType: Basics[Int64]}},
				{Name: "Write", Procedure: &Procedure{Function: false}},
			},
		}

		partial := &Struct{
			Methods: []*Method{
				{Name: "Read", Procedure: &Procedure{Function: true, ReturnType: Basics[Int64]}},
			},
		}

		if !Implements(full, readWrite) {
			t.Error("full should implement readWrite")
		}

		if Implements(partial, readWrite) {
			t.Error("partial should not implement readWrite")
		}
	})
}

func TestSatisfiesInterface(t *testing.T) {
	t.Parallel()

	proc := &Procedure{Function: true, ReturnType: Basics[UTF8]}
	stringer := &Interface{
		Methods: []*Method{{Name: "String", Procedure: proc}},
	}

	t.Run("struct_satisfies", func(t *testing.T) {
		t.Parallel()

		s := &Struct{
			Methods: []*Method{{Name: "String", Procedure: proc}},
		}

		if !Satisfies(s, stringer) {
			t.Error("struct with matching method should satisfy interface constraint")
		}
	})

	t.Run("struct_does_not_satisfy", func(t *testing.T) {
		t.Parallel()

		s := &Struct{}
		if Satisfies(s, stringer) {
			t.Error("struct without method should not satisfy interface constraint")
		}
	})

	t.Run("alias_to_interface_constraint", func(t *testing.T) {
		t.Parallel()

		// Wrapping the interface in an alias (like a named constraint)
		aliasConstraint := &Alias{Name: "Stringer", Derived: stringer}
		s := &Struct{
			Methods: []*Method{{Name: "String", Procedure: proc}},
		}

		if !Satisfies(s, aliasConstraint) {
			t.Error("struct should satisfy alias-wrapped interface")
		}
	})
}
