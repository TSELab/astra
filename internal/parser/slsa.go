package parser

import (
	"os"
	"time"
)

type SlsaParser struct{}

func (p *SlsaParser) Parse(path string) (Normalized, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Normalized{}, err
	}
	print(b)

	n := Normalized{Source: "SLSA", NormalizedAt: time.Now().Unix()}

	return n, nil
}
