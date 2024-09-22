package asn1mib

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
	"golang.org/x/exp/maps"
)

//-----------------------

type macroInstance struct {
}

//-----------------------

type sequenceOfFields struct {
	source   Position
	elements []TypeDefinition
}

func (sof *sequenceOfFields) Source() Position {
	return sof.source
}

func (sof *sequenceOfFields) compile(ctx context.Context, md *macroDefintion, tokens *TokenList) error {
	sof.source = *tokens.Source()
	for !tokens.IsEOF() {
		peek, _ := tokens.LookAhead(0)
		if peek.IsText("|") {
			return nil
		}
		tok, _ := tokens.Pop()

		switch tok.Type() {
		case IDENT:
			switch tok.String() {
			case "type":
				peek, err := tokens.LookAhead(0)
				if err != nil {
					return err
				}
				if peek.IsText("(") {
					block, err := tokens.PopBlock("(", ")")
					if err != nil {
						return err
					}
					if len(block.elements) != 1 {
						return block.Source().Errorf("expected single type")
					}
					reader, err := Lookup[TypeDefinition](ctx, block.elements[0].String())
					if err != nil {
						return block.Source().WrapError(err)
					}
					sof.elements = append(sof.elements, reader)
				} else {
					sof.elements = append(sof.elements, &simpleTypeDefintion{typeClass: "TYPE?", source: *tok.Source()})
				}
			case "value":
				block, err := tokens.PopBlock("(", ")")
				if err != nil {
					return err
				}
				if len(block.elements) < 1 {
					return block.Source().Errorf("expected value")
				}
				ident, err := block.Pop()
				if err != nil {
					return err
				}
				peek, _ := block.LookAhead(0)
				if peek != nil && peek.Type() == IDENT {
					ident, err = block.Pop()
					if err != nil {
						return err
					}
				}
				reader, err := Lookup[TypeDefinition](ctx, ident.String())
				if err != nil {
					return block.Source().WrapError(err)
				}
				sof.elements = append(sof.elements, reader)
			case "empty":
				sof.elements = append(sof.elements, &ConstantValue{source: *tok.Source()})
			case "identifier":
				sof.elements = append(sof.elements, &simpleTypeDefintion{typeClass: "IDENTIFIER", source: *tok.Source()})
			case "number":
				sof.elements = append(sof.elements, &simpleTypeDefintion{typeClass: "INTEGER", source: *tok.Source()})
			default:
				other, err := Lookup[TypeDefinition](ctx, tok.String())
				if err != nil {
					return tok.WrapError(err)
				}
				sof.elements = append(sof.elements, other)
			}
		case STRING:
			text, err := UnquoteString(tok)
			if err != nil {
				return err
			}
			sof.elements = append(sof.elements, &ConstantValue{Value: text, source: *tok.Source()})
		default:
			return tokens.Source().Errorf("compiling found unexpected token type %s %s", tok.Type(), tok)
		}
	}
	return nil
}

func (sof *sequenceOfFields) Read(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {
	if len(sof.elements) == 1 {
		return sof.elements[0].Read(ctx, name, meta, s)
	}
	return nil, sof.source.WrapError(asn1core.NewUnimplementedError("sequence of fields").MaybeLater())
}

type fieldDefintion struct {
	name       string
	source     Position
	tokens     *TokenList
	alternates []TypeDefinition
}

func (fd *fieldDefintion) compile(ctx context.Context, md *macroDefintion) error {

	if fd.alternates != nil {
		return nil
	}

	tokens := fd.tokens.Clone()

	seq := &sequenceOfFields{}
	err := seq.compile(ctx, md, tokens)
	if err != nil {
		return err
	}
	alternates := []TypeDefinition{seq}
	for !tokens.IsEOF() {
		peek, _ := tokens.LookAhead(0)
		if peek.IsText("|") {
			tokens.Pop()
			seq := &sequenceOfFields{}
			err := seq.compile(ctx, md, tokens)
			if err != nil {
				return err
			}
			alternates = append(alternates, seq)
		} else {
			break
		}
	}

	if !tokens.IsEOF() {
		return tokens.Source().Errorf("while compiling unexpected tokens %s", tokens)
	}

	fd.alternates = alternates

	return nil
}

func (fd *fieldDefintion) Read(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {
	if len(fd.alternates) == 1 {
		return fd.alternates[0].Read(ctx, name, meta, s)
	}
	return nil, fd.source.WrapError(asn1core.NewUnimplementedError("sequence of fields").MaybeLater())
}

func (fd *fieldDefintion) Source() Position {
	return fd.source
}

type macroDefintion struct {
	name     string
	source   Position
	fields   map[string]*fieldDefintion
	compiled bool
}

func (md *macroDefintion) Source() Position {
	return md.source
}

func (md *macroDefintion) Read(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {

	if !md.compiled {
		err := md.compile(ctx)
		if err != nil {
			return nil, err
		}
		md.compiled = true
	}

	if meta.Length() > 0 {
		typeNotation := md.fields["TYPE NOTATION"]
		if typeNotation == nil {
			return nil, meta.Source().Errorf("unexpected meta data %s", meta)
		}
	}

	valueNotation := md.fields["VALUE NOTATION"]
	if valueNotation == nil {
		return nil, meta.Source().Errorf("missing value notation")
	}

	definition, err := valueNotation.Read(ctx, name, meta, s)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (md *macroDefintion) compile(ctx context.Context) error {
	todo := maps.Keys(md.fields)
	errors := asn1core.ErrorList{}

	ctx = withContext(ctx, func(ctx context.Context, name string) (Definition, error) {
		def, ok := md.fields[name]
		if !ok {
			return nil, fmt.Errorf("unknown definition %s", name)
		}
		return def, nil
	})

	for len(todo) > 0 {
		var failed []string
		errors = errors[:0] //truncate
		progress := false
		for _, name := range todo {
			err := md.fields[name].compile(ctx, md)
			if err != nil {
				failed = append(failed, name)
				errors = append(errors, err)
			} else {
				progress = true
			}
		}
		if !progress {
			break
		}
		todo = failed
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func (md *macroDefintion) Initialize(ctx context.Context, name string, meta *TokenList, s *Scanner) error {
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
			field, err := readFieldDefintion(ctx, fieldTokens)
			if err != nil {
				return err
			}
			md.fields[field.name] = field
			startBlock = i
		}
	}

	fieldTokens, _ := block.Slice(startBlock, len(block.elements))
	field, err := readFieldDefintion(ctx, fieldTokens)
	if err != nil {
		return err
	}
	md.fields[field.name] = field

	return nil
}

func readFieldDefintion(ctx context.Context, tokens *TokenList) (*fieldDefintion, error) {
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
	return &fieldDefintion{name: name.String(), tokens: tokens, source: *name.Source()}, nil
}
