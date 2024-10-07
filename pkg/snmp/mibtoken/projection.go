package mibtoken

type Projection struct {
	reader Reader
	source Source
	offset int
}

func NewProjection(reader Reader) *Projection {
	return &Projection{reader: reader, source: *reader.Source()}
}

var _ Reader = (*Projection)(nil)

func (p *Projection) Pop() (*Token, error) {
	tok, err := p.LookAhead(0)
	if err != nil {
		return tok, err
	}
	p.offset++
	return tok, nil
}

func (p *Projection) LookAhead(n int) (*Token, error) {
	tok, err := p.reader.LookAhead(p.offset + n)
	return tok, err
}

func (p *Projection) IsEOF() bool {
	peek, err := p.LookAhead(0)
	if err != nil || peek == nil || peek.Type() == EOF {
		return true
	}
	return peek == nil
}

func (p *Projection) Source() *Source {
	peek, err := p.LookAhead(0)
	if err != nil {
		return &p.source
	}
	return peek.Source()
}

func (p *Projection) Commit() {
	for p.offset > 0 {
		p.reader.Pop()
		p.offset--
	}
}
