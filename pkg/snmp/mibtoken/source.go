package mibtoken

import (
	"fmt"
	"strings"
)

type Source struct {
	Filename string
	Line     int
	Column   int
}

func (p *Source) String() string {
	sb := strings.Builder{}
	if p.Filename != "" {
		sb.WriteString(p.Filename)
		if p.Line != 0 || p.Column != 0 {
			sb.WriteString(" ")
		}
	}
	if p.Line > 0 || p.Column > 0 {
		sb.WriteString(fmt.Sprintf("[ln %d col %d]", p.Line, p.Column))
	}
	if p.Line <= 0 && p.Column <= 0 {
		sb.WriteString("[EOF]")
	}

	return sb.String()
}

func (p *Source) IsEOF() bool {
	return p.Line <= 0 && p.Column <= 0
}

func EOFPosition(filename string) *Source {
	return &Source{Filename: filename}
}
