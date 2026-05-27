package builtin

func If[T any, B ~bool](condition B, consequence T, alternative ...T) T {
	if condition {
		return consequence
	}

	if len(alternative) == 0 {
		return *new(T)
	}

	return alternative[0]
}
