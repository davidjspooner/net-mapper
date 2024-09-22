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

func (u *simpleTypeDefintion) ReadDefinition(name string, d *Directory, s *Scanner) (Definition, error) {
	return nil, u.source.WrapError(asn1core.NewUnimplementedError("simple type definition %s", name).MaybeLater())
}

// -----------------------

type simpleTypeDefintionReader struct {
}

var closer = map[string]string{"{": "}", "(": ")", "[": "]"}

func (u *simpleTypeDefintionReader) ReadDefinition(name string, d *Directory, s *Scanner) (Definition, error) {
	pos := s.LookAhead(0).source
	meta, err := s.PopUntil("::=")
	_ = meta
	if err != nil {
		return nil, pos.Errorf("unterminated definition %q", name)
	}
	ident := s.LookAhead(0)
	td := &simpleTypeDefintion{source: ident.source}

	if ident.TokenIs("[") {
		block, err := s.PopBlock(1, "[", "]")
		if err != nil {
			return nil, err
		}
		if len(block) != 3 {
			return nil, block[0].source.Errorf("unexpected block %s", block)
		}
		class, err := asn1core.ParseClass(block[0].String())
		if err != nil {
			return nil, block[0].WrapError(err)
		}
		td.params.Class = asn1binary.PtrToClass(class)
		tag, err := strconv.Atoi(block[1].String())
		td.params.Tag = asn1binary.PtrToTag(asn1core.Tag(tag))
		if err != nil {
			return nil, block[1].WrapError(err)
		}
		//TODO set the tags?
		peek := s.LookAhead(1)
		if peek.TokenIs("IMPLICIT") {
			s.Scan()
			td.implicit = true
		}
		ident, err = s.PopIdent()
		if err != nil {
			return nil, err
		}
	}

	td.typeClass = ident.String()
	switch ident.String() {
	case Octet_String, "INTEGER", "CHOICE", "SEQUENCE", "SEQUENCE OF", "SET", "SET OF":
		peek := s.LookAhead(1)
		open := peek.String()
		close, isBracket := closer[open]
		if isBracket {
			constraint, err := s.PopBlock(0, open, close)
			if err != nil {
				return nil, err
			}
			td.constraint = constraint[1 : len(constraint)-1] //strip the ")"
			return td, nil
		}
		return td, nil
	case Object_Identifier:
		peek := s.LookAhead(1)
		if peek.Type() == IDENT {
			return td, nil
		}
		return nil, td.source.WrapError(asn1core.NewUnimplementedError("Object_Identifier type definition").MaybeLater())
	default:
		reader, ok := d.definitions[ident.String()]
		if !ok {
			return nil, td.source.WrapError(asn1core.NewUnimplementedError("unknown type definition %s", ident.String()).MaybeLater())
		}
		defReader, ok := reader.(TypeDefinition)
		if !ok {
			return nil, td.source.WrapError(asn1core.NewUnimplementedError("Type definition %s is not a reader", ident.String()).MaybeLater())
		}
		def, err := defReader.Read(name, d, s)
		if err != nil {
			return nil, err
		}
		return def, nil
	}

}

func (reader *simpleTypeDefintionReader) Source() Position {
	return builtInPosition
}
