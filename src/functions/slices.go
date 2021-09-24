package functions

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func IsContain(slice []string, val string) bool {
	_, result := Find(slice, val)
	return result
}
