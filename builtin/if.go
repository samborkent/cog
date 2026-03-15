package builtin

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
