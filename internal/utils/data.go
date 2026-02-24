package utils

func RemoveDuplicates[T comparable](sliceList []T) []T {
	keys := make(map[T]bool)
	l := []T{}
	for _, item := range sliceList {
		if _, value := keys[item]; !value {
			keys[item] = true
			l = append(l, item)
		}
	}
	return l
}
