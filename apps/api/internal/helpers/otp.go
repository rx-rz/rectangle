package helpers

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type Generator struct {
	length int
}

func NewGenerator(length int) *Generator {
	return &Generator{length: length}
}

func (g *Generator) Generate() (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(g.length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	format := fmt.Sprintf("%%0%dd", g.length)

	return fmt.Sprintf(format, n), nil
}

func GenerateOTP() (string, error) {
	gen := NewGenerator(6)
	return gen.Generate()
}
