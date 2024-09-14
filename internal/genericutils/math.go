package genericutils

import "golang.org/x/exp/constraints"

func Max[T constraints.Ordered](a T, others ...T) T {
	max := a
	for _, v := range others {
		if v > max {
			max = v
		}
	}
	return max
}

func Min[T constraints.Ordered](a T, others ...T) T {
	min := a
	for _, v := range others {
		if v < min {
			min = v
		}
	}
	return min
}
