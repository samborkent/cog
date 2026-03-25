package builtin

import "golang.org/x/exp/constraints"

type Real interface {
	constraints.Integer | constraints.Float
}

func Range[T Real](from, to T, steps ...int) []T {
	to = max(from, to)

	numSteps := 1

	if len(steps) > 0 {
		numSteps = max(steps[0], numSteps)
	}

	stepSize := 1 / T(numSteps)
	length := (from - to) * stepSize

	slice := make([]T, int(length))

	for i := range numSteps {
		slice[i] = from + T(i)*stepSize
	}

	return slice
}
