package util

func BoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
