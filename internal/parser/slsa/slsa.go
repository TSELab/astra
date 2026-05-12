package slsa

import (
	"io"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
)

type SlsaParser struct{}

func (p *SlsaParser) Parse(r io.Reader) (parser.Evidence, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return parser.Evidence{}, err
	}
	print(b)

	n := parser.Evidence{Source: "SLSA", NormalizedAt: time.Now().Unix()}

	return n, nil
}
