package cog

import (
	"reflect"
	"testing"
)

func TestCopy(t *testing.T) {
	t.Parallel()

	t.Run("basic_types_no_copy", func(t *testing.T) {
		t.Parallel()

		// Test int
		t.Run("int", func(t *testing.T) {
			t.Parallel()
			original := 42
			result := Copy(original)
			if result != original {
				t.Errorf("Copy(%v) = %v, want %v", original, result, original)
			}
		})

		// Test string
		t.Run("string", func(t *testing.T) {
			t.Parallel()
			original := "hello"
			result := Copy(original)
			if result != original {
				t.Errorf("Copy(%v) = %v, want %v", original, result, original)
			}
		})

		// Test bool
		t.Run("bool", func(t *testing.T) {
			t.Parallel()
			original := true
			result := Copy(original)
			if result != original {
				t.Errorf("Copy(%v) = %v, want %v", original, result, original)
			}
		})

		// Test float64
		t.Run("float64", func(t *testing.T) {
			t.Parallel()
			original := 3.14
			result := Copy(original)
			if result != original {
				t.Errorf("Copy(%v) = %v, want %v", original, result, original)
			}
		})
	})

	t.Run("slice_basic_type", func(t *testing.T) {
		t.Parallel()
		original := []int{1, 2, 3, 4, 5}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Should be different instances
		if len(copied) > 0 && &copied[0] == &original[0] {
			t.Error("Copy should create new slice instance")
		}

		// Modifying copy should not affect original
		copied[0] = 999
		if original[0] != 1 {
			t.Error("Modifying copy affected original slice")
		}
	})

	t.Run("slice_struct_type", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		original := []Person{
			{"Alice", 30},
			{"Bob", 25},
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Should be different instances
		if len(copied) > 0 && &copied[0] == &original[0] {
			t.Error("Copy should create new slice instance")
		}

		// Modifying copy should not affect original
		copied[0].Name = "Charlie"
		if original[0].Name != "Alice" {
			t.Error("Modifying copy affected original slice")
		}
	})

	t.Run("map_basic_type", func(t *testing.T) {
		t.Parallel()
		original := map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Modifying copy should not affect original
		copied["a"] = 999
		if original["a"] != 1 {
			t.Error("Modifying copy affected original map")
		}

		// Adding to copy should not affect original
		copied["d"] = 4
		if _, exists := original["d"]; exists {
			t.Error("Adding to copy affected original map")
		}
	})

	t.Run("map_struct_type", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		original := map[string]Person{
			"alice": {"Alice", 30},
			"bob":   {"Bob", 25},
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Modifying copy should not affect original
		person := copied["alice"]
		person.Name = "Charlie"
		copied["alice"] = person
		if original["alice"].Name != "Alice" {
			t.Error("Modifying copy affected original map")
		}
	})

	t.Run("struct_with_pointer_fields", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name    string
			Age     int
			Address *string
		}

		address := "123 Main St"
		original := Person{
			Name:    "Alice",
			Age:     30,
			Address: &address,
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Pointer field should be different instance
		if copied.Address == original.Address {
			t.Error("Copy should create new pointer instance")
		}

		// Modifying copy should not affect original
		*copied.Address = "456 Oak Ave"
		if *original.Address != "123 Main St" {
			t.Error("Modifying copy affected original struct")
		}
	})

	t.Run("struct_no_pointer_fields", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		original := Person{
			Name: "Alice",
			Age:  30,
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Modifying copy should not affect original
		copied.Name = "Bob"
		if original.Name != "Alice" {
			t.Error("Modifying copy affected original")
		}
	})

	t.Run("nested_struct", func(t *testing.T) {
		t.Parallel()

		type Address struct {
			Street string
			City   string
		}

		type Person struct {
			Name    string
			Address Address
		}

		original := Person{
			Name: "Alice",
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Modifying nested struct should not affect original
		copied.Address.Street = "456 Oak Ave"
		if original.Address.Street != "123 Main St" {
			t.Error("Modifying nested copy affected original")
		}
	})

	t.Run("pointer_to_struct", func(t *testing.T) {
		t.Parallel()

		type Person struct {
			Name string
			Age  int
		}

		original := &Person{
			Name: "Alice",
			Age:  30,
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Should be different pointer
		if copied == original {
			t.Error("Copy should create new pointer")
		}

		// Modifying copy should not affect original
		copied.Name = "Bob"
		if original.Name != "Alice" {
			t.Error("Modifying copy affected original")
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		t.Parallel()

		original := []int{}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Should be different instances (but both empty)
		if &copied == &original {
			t.Error("Copy should create new slice instance even for empty slices")
		}
	})

	t.Run("nil_slice", func(t *testing.T) {
		t.Parallel()

		var original []int
		copied := Copy(original)

		// Should have same values (both nil)
		if copied != nil {
			t.Errorf("Copy(nil slice) = %v, want nil", copied)
		}
	})

	t.Run("complex_nested_structure", func(t *testing.T) {
		t.Parallel()

		type Address struct {
			Street string
			City   string
		}

		type Person struct {
			Name     string
			Age      int
			Address  *Address
			Friends  []string
			Metadata map[string]interface{}
		}

		original := Person{
			Name: "Alice",
			Age:  30,
			Address: &Address{
				Street: "123 Main St",
				City:   "New York",
			},
			Friends: []string{"Bob", "Charlie"},
			Metadata: map[string]interface{}{
				"role":   "developer",
				"skills": []string{"go", "python"},
			},
		}
		copied := Copy(original)

		// Should have same values
		if !reflect.DeepEqual(copied, original) {
			t.Errorf("Copy(%v) = %v, want %v", original, copied, original)
		}

		// Test deep copy by modifying various parts
		copied.Name = "Bob"
		if original.Name != "Alice" {
			t.Error("Modifying copied.Name affected original")
		}

		copied.Address.Street = "456 Oak Ave"
		if original.Address.Street != "123 Main St" {
			t.Error("Modifying copied.Address affected original")
		}

		copied.Friends[0] = "Dave"
		if original.Friends[0] != "Bob" {
			t.Error("Modifying copied.Friends affected original")
		}

		copied.Metadata["role"] = "manager"
		if original.Metadata["role"] != "developer" {
			t.Error("Modifying copied.Metadata affected original")
		}

		skills := copied.Metadata["skills"].([]string)
		skills[0] = "java"
		if original.Metadata["skills"].([]string)[0] != "go" {
			t.Error("Modifying nested slice in copied.Metadata affected original")
		}
	})
}
