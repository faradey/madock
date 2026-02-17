package hash

import (
	"crypto/rand"
	"math/big"
	"strings"
)

var (
	lowerCharSet   = "abcdedfghijklmnopqrst"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

// PasswordGenerator allows enterprise to provide a custom password generation strategy.
type PasswordGenerator func(length int) (string, error)

var passwordGenerator PasswordGenerator

// SetPasswordGenerator sets a custom password generator.
func SetPasswordGenerator(gen PasswordGenerator) {
	passwordGenerator = gen
}

func GeneratePassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	if passwordGenerator != nil {
		if pw, err := passwordGenerator(passwordLength); err == nil {
			return pw
		}
	}

	var password strings.Builder

	for i := 0; i < minSpecialChar; i++ {
		password.WriteByte(specialCharSet[cryptoRandIntn(len(specialCharSet))])
	}

	for i := 0; i < minNum; i++ {
		password.WriteByte(numberSet[cryptoRandIntn(len(numberSet))])
	}

	for i := 0; i < minUpperCase; i++ {
		password.WriteByte(upperCharSet[cryptoRandIntn(len(upperCharSet))])
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		password.WriteByte(allCharSet[cryptoRandIntn(len(allCharSet))])
	}

	inRune := []rune(password.String())
	cryptoShuffle(inRune)
	return string(inRune)
}

// cryptoRandIntn returns a cryptographically secure random int in [0, n).
func cryptoRandIntn(n int) int {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return int(val.Int64())
}

// cryptoShuffle performs a Fisher-Yates shuffle using crypto/rand.
func cryptoShuffle(s []rune) {
	for i := len(s) - 1; i > 0; i-- {
		j := cryptoRandIntn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}
