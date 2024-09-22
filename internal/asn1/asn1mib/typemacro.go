package asn1mib

import (
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

//-----------------------

type macroInstance struct {
}

//-----------------------

type fieldDefintion struct {
	name   string
	tokens *TokenList
}

type macroDefintion struct {
	name   string
	source Position
	fields map[string]*fieldDefintion
}

func (md *macroDefintion) Source() Position {
	return md.source
}

func (md *macroDefintion) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return nil, err
	}
	_ = meta
	return nil, md.source.WrapError(asn1core.NewUnimplementedError("macro usage - %s %s", name, md.name).MaybeLater())
}

func (md *macroDefintion) Initialize(name string, d *Directory, s *Scanner) error {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return err
	}
	_ = meta
	block, err := s.PopBlock("BEGIN", "END")
	if err != nil {
		return err
	}
	md.name = name
	md.source = *block.Source()
	md.fields = make(map[string]*fieldDefintion)

	startBlock := 0
	for i := startBlock + 2; i < (block.Length() - 1); i++ {
		peek1, _ := block.LookAhead(i + 1)
		if peek1.IsText("::=") {
			fieldTokens, _ := block.Slice(startBlock, i)
			field, err := readFieldDefintion(fieldTokens)
			if err != nil {
				return err
			}
			md.fields[field.name] = field
			startBlock = i
		}
	}

	fieldTokens, _ := block.Slice(startBlock, len(block.elements))
	field, err := readFieldDefintion(fieldTokens)
	if err != nil {
		return err
	}
	md.fields[field.name] = field
	return nil
}

func readFieldDefintion(tokens *TokenList) (*fieldDefintion, error) {
	if tokens.Length() < 3 {
		return nil, tokens.Source().Errorf("unexpected field definition %s", tokens)
	}
	name, err := tokens.Pop()
	if err != nil {
		return nil, err
	}
	if name.Type() != IDENT {
		return nil, name.Errorf("expected field name")
	}
	err = tokens.PopExpected("::=")
	if err != nil {
		return nil, err
	}
	fmt.Println("field tokens", name, "=", tokens)
	return &fieldDefintion{name: name.String(), tokens: tokens}, nil
}
