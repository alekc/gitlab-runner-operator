package generate

import (
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
)

// RandomString returns a random string of a required length
func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
