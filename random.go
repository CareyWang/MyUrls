package main

import (
	"math/rand"
	"time"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// generate is a function that takes an integer bits and returns a string.
// The function generates a random string of length equal to bits using the letterBytes slice.
// The letterBytes slice contains characters that can be used to generate a random string.
// The generation of the random string is based on the current time using the UnixNano() function.
func GenerateRandomString(bits int) string {
	// Create a byte slice b of length bits.
	b := make([]byte, bits)

	// Create a new random number generator with the current time as the seed.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a random byte for each element in the byte slice b using the letterBytes slice.
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}

	// Convert the byte slice to a string and return it.
	return string(b)
}
