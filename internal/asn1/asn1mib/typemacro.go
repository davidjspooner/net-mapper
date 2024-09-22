package asn1mib

import (
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

//-----------------------

type macroInstance struct {
}

//-----------------------

type fieldDefintion struct {
	name   string
	tokens TokenList
}

type macroDefintion struct {
	name   string
	source Position
	fields []fieldDefintion
}

func (md *macroDefintion) Source() Position {
	return md.source
}

func (md *macroDefintion) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return nil, err
	}
	meta = meta[:len(meta)-1] //strip the ::= token
	_ = meta
	return nil, md.source.WrapError(asn1core.NewUnimplementedError("macro usage - %s %s", name, md.name).MaybeLater())
}

type macroDefintionReader struct {
}

func (mdr *macroDefintionReader) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return nil, err
	}
	_ = meta
	pos := s.LookAhead(0).source
	block, err := s.PopBlock(0, "BEGIN", "END")
	if err != nil {
		return nil, err
	}
	block = block[1 : len(block)-1] //strip the BEGIN and END tokens
	md := &macroDefintion{source: pos, name: name}

	startBlock := 0
	for i := startBlock + 2; i < (len(block) - 1); i++ {
		if block[i+1].TokenIs("::=") {
			md.fields = append(md.fields, fieldDefintion{name: block[startBlock].String(), tokens: block[startBlock+2 : i]})
			startBlock = i
		}
	}
	md.fields = append(md.fields, fieldDefintion{name: block[startBlock].String(), tokens: block[startBlock+2:]})

	return md, nil
}

func (mdr *macroDefintionReader) Source() Position {
	return builtInPosition
}
