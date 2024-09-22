package asn1mib

import (
	"strconv"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1binary"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

//-----------------------

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

func (u *simpleTypeDefintion) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	return nil, u.source.WrapError(asn1core.NewUnimplementedError("simple type definition %s", name).MaybeLater())
}

// -----------------------

var closer = map[string]string{"{": "}", "(": ")", "[": "]"}

func (td *simpleTypeDefintion) Initialize(name string, d *Directory, s *Scanner) error {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return meta.Source().Errorf("unterminated definition %q", name)
	}
	_ = meta
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
	switch ident.String() {
	case Octet_String, "INTEGER", "CHOICE", "SEQUENCE", "SEQUENCE OF", "SET", "SET OF":
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
	case Object_Identifier:
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		if peek.Type() == IDENT {
			return nil
		}
		return td.source.WrapError(asn1core.NewUnimplementedError("Object_Identifier type definition").MaybeLater())
	default:
		return td.source.WrapError(asn1core.NewUnimplementedError("simple type definition %s", ident.String()).MaybeLater())
	}
}
