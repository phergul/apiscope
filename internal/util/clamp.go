package util

import "cmp"

// Clamp bounds an ordered value within an inclusive minimum and maximum range.
func Clamp[T cmp.Ordered](value, minimum, maximum T) T {
	return min(max(value, minimum), maximum)
}
