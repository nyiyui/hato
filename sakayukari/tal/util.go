package tal

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func reverseMap[K comparable, V comparable](m map[K]V) map[V]K {
	res := map[V]K{}
	for key, value := range m {
		res[value] = key
	}
	return res
}

func abs[T int | int64](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
