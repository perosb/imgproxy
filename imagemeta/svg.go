package imagemeta

import (
	"io"
	"strings"

	"github.com/imgproxy/imgproxy/v3/config"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/xml"
)

func IsSVG(r io.Reader) (bool, error) {
	maxBytes := config.MaxSvgCheckBytes

	l := xml.NewLexer(parse.NewInput(io.LimitReader(r, int64(maxBytes))))

	for {
		tt, _ := l.Next()

		switch tt {
		case xml.ErrorToken:
			if err := l.Err(); err != io.EOF {
				return false, err
			}
			return false, nil

		case xml.StartTagToken:
			return strings.ToLower(string(l.Text())) == "svg", nil
		}
	}
}
