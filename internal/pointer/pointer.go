package pointer

// StringSlice returns a pointer to a slice of strings
func StringSlice(input []string) *[]string {
	return &input
}

// String returns a pointer to a string.
func String(s string) *string {
	return &s
}
