package file

import "os"

// Exist quick and dirty check if file exists, and it's readable
func Exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
