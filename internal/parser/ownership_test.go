package parser_test

import (
	"testing"
)

func TestOwnership(t *testing.T) {
	t.Parallel()

	t.Run("main_func_borrow_after_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
main_func_borrow : proc() = {
	x : var []int64 = @slice<int64>(3)
	y := len(x)
	@print(y)
}
len : func(x : []int64) int64 = { return 0 }`)
	})

	t.Run("main_proc_consume_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_proc_consume : proc() = {
	x : var []int64 = @slice<int64>(3)
	takes_slice(x)
	@print(x)
}
takes_slice : proc(x : []int64) = {}`)
	})

	t.Run("main_var_to_var_consume_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_var_to_var : proc() = {
	x : var []int64 = @slice<int64>(3)
	y : var []int64 = x
	@print(x)
}`)
	})

	t.Run("main_ref_consumes_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_ref_consumes : proc() = {
	x : var int64 = 42
	r := &x
	@print(x)
}`)
	})

	t.Run("main_deref_consumes_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_deref_consumes : proc() = {
	x : var int64 = 42
	r : var &int64 = &x
	s := *r
	@print(r)
}`)
	})

	t.Run("main_defer_double_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_defer_double : proc() = {
	x : var int64 = 42
	defer echo(x)
	defer echo(x)
}
echo : proc(x : int64) = { @print(x) }`)
	})

	t.Run("main_defer_already_dead_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_defer_dead : proc() = {
	x : var int64 = 42
	y := x
	defer echo(x)
}
echo : proc(x : int64) = { @print(x) }`)
	})

	t.Run("main_cond_partial_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_cond_partial : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	if true {
		d := c.data
		@print(d[0])
	} else {
		@print(c.data[0])
	}
	@print(c.label)
}`)
	})

	t.Run("main_for_borrow_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
main_for_borrow : proc() = {
	items : var []int64 = []int64{1, 2, 3}
	for item in items {
		@print(item)
	}
	@print(items[0])
}`)
	})

	t.Run("main_map_key_var_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_map_key_var : proc() = {
	m : var map[utf8]var []int64 = @map<utf8, []int64>()
	key : utf8 = "items"
	val : var []int64 = @slice<int64>(3)
	m[key] = val
	@print(val)
}`)
	})

	t.Run("main_map_key_not_comparable_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_map_key_nc : proc() = {
	m : map[[]int64]utf8
}`)
	})

	t.Run("main_set_element_not_comparable_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_set_nc : proc() = {
	s : set[[]int64]
}`)
	})

	t.Run("main_func_var_param_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
bad : func(x : var int64) int64 = { return x }`)
	})

	t.Run("main_switch_consume_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_switch_consume : proc() = {
	x : var []int64 = @slice<int64>(3)
	switch {
	case true:
		d := x
		@print(d[0])
	case false:
		@print(x)
	}
	@print(x)
}`)
	})

	t.Run("main_index_borrow_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
main_index_borrow : proc() = {
	items : var []int64 = []int64{1, 2, 3}
	x := items[0]
	@print(x)
	@print(items[0])
}`)
	})

	t.Run("main_enum_borrow_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
Result ~ enum<int8> {
	Ok := 1;
	Err := 2
}
main_enum_borrow : proc() = {
	r : var Result = Result.Ok
	@print(r)
}`)
	})

	t.Run("main_defer_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
main_defer_ok : proc() = {
	x : var int64 = 42
	defer echo(x)
	@print(x + 1)
}
echo : proc(x : int64) = { @print(x) }`)
	})

	t.Run("main_return_dead_field_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_return_dead : proc() Container = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	d := c.data
	@print(d[0])
	return c
}`)
	})

	t.Run("main_revive_partial_move_reassign_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_revive : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	d := c.data
	@print(d[0])
	c.data = @slice<int64>(5)
	use_container(c)
}
use_container : proc(c : Container) = {}`)
	})

	t.Run("main_defer_field_after_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_defer_field_after : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	defer print_data(c.data)
	c.data = @slice<int64>(5)
}
print_data : proc(d : []int64) = {}`)
	})

	t.Run("main_return_partial_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_return_partial : proc() Container = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	d := c.data
	@print(d[0])
	c.data = @slice<int64>(5)
	return c
}`)
	})

	// Issue 2: Defer on a variable that was already consumed before the defer expression.
	t.Run("main_defer_already_consumed_before_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_defer_pre_consumed : proc() = {
	x : var []int64 = @slice<int64>(3)
	takes_slice(x)
	defer echo(x)
}
echo : proc(x : int64) = {}
takes_slice : proc(x : []int64) = {}`)
	})

	// Issue 4: Proc call on a deferred-reserved variable should be rejected.
	t.Run("main_defer_then_proc_consume_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_defer_then_proc : proc() = {
	x : var []int64 = @slice<int64>(3)
	defer echo_len(x)
	takes_slice(x)
}
echo_len : proc(x : []int64) = { @print(len(x)) }
takes_slice : proc(x : []int64) = {}`)
	})

	// Issue 4: Field move-out on a deferred-reserved field should be rejected.
	t.Run("main_defer_then_field_move_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_defer_then_move : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	defer print_data(c.data)
	d := c.data
}
print_data : proc(d : []int64) = {}`)
	})

	// Issue 9: Func borrow on a partially-moved struct should be rejected.
	t.Run("main_func_borrow_partial_move_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_func_borrow_partial : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	d := c.data
	@print(d[0])
	read_container(c)
}
read_container : func(c : Container) = {}`)
	})

	// Issue 9: Func borrow on a fully alive struct works after re-assignment.
	t.Run("main_func_borrow_after_revive_ok", func(t *testing.T) {
		t.Parallel()
		parse(t, `package p
Container ~ struct { data : []int64; label : utf8 }
main_func_revive_borrow : proc() = {
	c : var Container = Container{data = @slice<int64>(3), label = "hello"}
	d := c.data
	@print(d[0])
	c.data = @slice<int64>(5)
	read_container(c)
}
read_container : func(c : Container) = {}`)
	})

	// Var to var assignment of primitive still consumes (Rule 4 applies uniformly).
	t.Run("main_var_to_var_primitive_consume_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main_var_to_var_prim : proc() = {
	x : var int64 = 42
	y : var int64 = x
	@print(x)
}`)
	})
}