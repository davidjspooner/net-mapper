package asn1mib

import (
	"strconv"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1go"
)

// -----------------------
type OIDValue interface {
	OID() asn1go.OID
}
type oidDefintion struct {
	oid    asn1go.OID
	source Position
}

func (o *oidDefintion) OID() asn1go.OID {
	return o.oid
}
func (o *oidDefintion) Source() Position {
	return o.source
}

type oidReader struct {
}

func (o *oidReader) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	meta, err := s.PopUntil("::=")
	if err != nil {
		return nil, err
	}
	_ = meta
	tokens, err := s.PopBlock("{", "}")
	if err != nil {
		return nil, err
	}
	valuePosition := tokens.Source()
	if tokens.IsEOF() {
		return nil, valuePosition.Errorf("empty OID definition")
	}
	var oid asn1go.OID
	for i := 0; i < tokens.Length(); i++ {
		token, _ := tokens.LookAhead(i)
		switch token.Type() {
		case NUMBER:
			tail, err := asn1go.ParseOID(token.String(), d.OIDLookup)
			if err != nil {
				return nil, err
			}
			oid = append(oid, tail...)
		case IDENT:

			peek1, _ := tokens.LookAhead(i + 1)
			peek2, _ := tokens.LookAhead(i + 2)
			peek3, _ := tokens.LookAhead(i + 3)

			if tokens.Length() > i+3 && peek1.IsText("(") && peek3.IsText(")") {
				n, err := strconv.Atoi(peek2.String())
				if err != nil {
					return nil, peek2.WrapError(err)
				}
				oid = append(oid, n)
				i += 3
			} else {
				ref, ok := d.definitions[token.String()]
				if !ok {
					return nil, token.Errorf("unknown reference %s", token.String())
				}
				oidDefintion, ok := ref.(OIDValue)
				if !ok {
					return nil, token.Errorf("reference %s is not an OID", token.String())
				}
				oid = nil
				oid = append(oid, oidDefintion.OID()...)
			}
		default:
			return nil, valuePosition.Errorf("unexpected token %s", token.String())
		}
	}

	return &oidDefintion{oid: oid, source: *valuePosition}, nil
}

func (reader *oidReader) Source() Position {
	return builtInPosition
}
