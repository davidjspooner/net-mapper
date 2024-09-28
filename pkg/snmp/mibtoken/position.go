package mibtoken

import (
	"fmt"
	"strings"
)

type Position struct {
	Filename string
	Line     int
	Column   int
}

func (p *Position) String() string {
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

func (p *Position) IsEOF() bool {
	return p.Line <= 0 && p.Column <= 0
}

func EOFPosition(filename string) *Position {
	return &Position{Filename: filename}
}
