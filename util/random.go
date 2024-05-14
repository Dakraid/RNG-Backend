package util

import (
	"crypto/rand"
	"log"
	"math/big"
)

func Intn(max int64) int64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		log.Fatal(err)
	}
	return nBig.Int64()
}

func RandomFloat64() float64 {
	return float64(Intn(1<<53)) / (1 << 53)
}
