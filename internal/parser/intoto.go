package parser

import (
	"os"
	"time"
)

type InTotoParser struct{}

func (p *InTotoParser) Parse(path string) (Normalized, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Normalized{}, err
	}
	print(b)

	n := Normalized{Source: "in-toto", NormalizedAt: time.Now().Unix()}
	return n, nil
}
