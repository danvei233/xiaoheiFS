package cryptox

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// RandomAlphabet returns a cryptographically uniform string of length n drawn
// from alphabet using crypto/rand.Int, avoiding the modulo bias of a
// byte%len(alphabet) mapping (F5). An empty alphabet or n<0 is an error; n==0
// returns an empty string.
func RandomAlphabet(alphabet string, n int) (string, error) {
	if alphabet == "" {
		return "", fmt.Errorf("cryptox: empty alphabet")
	}
	if n < 0 {
		return "", fmt.Errorf("cryptox: negative length")
	}
	if n == 0 {
		return "", nil
	}
	charset := []rune(alphabet)
	max := big.NewInt(int64(len(charset)))
	out := make([]rune, n)
	for i := range out {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = charset[v.Int64()]
	}
	return string(out), nil
}
