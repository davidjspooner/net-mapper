package asn1mib

import (
	"context"
	"slices"
	"strconv"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

//-----------------------

var simpleTypeNames = []string{"INTEGER", "OCTET STRING", "SEQUENCE", "SEQUENCE OF", "CHOICE", "OBJECT IDENTIFIER", "IA5String"}

type simpleValue struct {
	source Position
}

func (v *simpleValue) Source() Position {
	return v.source
}

type simpleTypeDefintion struct {
	typeClass  string
	constraint TokenList
	source     Position
	implicit   bool
	params     asn1binary.Parameters
}

func (td *simpleTypeDefintion) Source() Position {
	return td.source
}

func (td *simpleTypeDefintion) ReadOID(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {
	tokens, err := s.PopBlock("{", "}")
	if err != nil {
		return nil, err
	}
	valuePosition := tokens.Source()
	if tokens.IsEOF() {
		return nil, valuePosition.Errorf("empty OID definition")
	}
	return &oidDefintion{tokens: *tokens, source: *valuePosition}, nil
}

func (u *simpleTypeDefintion) Read(ctx context.Context, name string, meta *TokenList, s *Scanner) (Definition, error) {
	switch u.typeClass {
	case Object_Identifier:
		oid, err := u.ReadOID(ctx, name, meta, s)
		if err != nil {
			return nil, err
		}
		return oid, nil
	default:
		reader, err := Lookup[TypeDefinition](ctx, u.typeClass)
		if err != nil {
			return nil, u.source.WrapError(err)
		}
		return reader.Read(ctx, name, meta, s)
	}
}

// -----------------------

var closer = map[string]string{"{": "}", "(": ")", "[": "]"}

func (td *simpleTypeDefintion) Initialize(ctx context.Context, name string, meta *TokenList, s *Scanner) error {
	ident, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	td.source = *meta.Source()

	if ident.IsText("[") {
		block, err := s.PopBlock("[", "]")
		if err != nil {
			return err
		}
		if block.Length() != 2 {
			return block.Source().Errorf("unexpected block %s", block)
		}
		classTok, _ := block.LookAhead(0)
		class, err := asn1core.ParseClass(classTok.String())
		if err != nil {
			return classTok.WrapError(err)
		}
		td.params.Class = asn1binary.PtrToClass(class)
		tagTok, _ := block.LookAhead(1)
		tag, err := strconv.Atoi(tagTok.String())
		td.params.Tag = asn1binary.PtrToTag(asn1core.Tag(tag))
		if err != nil {
			return tagTok.WrapError(err)
		}

		//TODO set the tags?
		ident, err = s.LookAhead(0)
		if err != nil {
			return err
		}
		if ident.IsText("IMPLICIT") {
			s.Pop()
			td.implicit = true
		}
	}
	ident, err = s.PopType(IDENT)
	if err != nil {
		return err
	}

	td.typeClass = ident.String()
	if slices.Contains(simpleTypeNames, td.typeClass) {
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		open := peek.String()
		close, isBracket := closer[open]
		if isBracket {
			constraint, err := s.PopBlock(open, close)
			if err != nil {
				return err
			}
			td.constraint = *constraint
		}
		return nil
	}
	def, err := Lookup[TypeDefinition](ctx, ident.String())
	if err != nil {
		return ident.WrapError(err)
	}
	_ = def
	return ident.WrapError(asn1core.NewUnimplementedError("simple type definition %s", ident.String()).MaybeLater())
}
