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
	_ = meta //discard for now
	err = s.PopExpected("{")
	if err != nil {
		return nil, err
	}
	valuePosition := s.LookAhead(0).source
	tokens, err := s.PopUntil("}")
	if err != nil {
		return nil, err
	}
	tokens = tokens[1 : len(tokens)-1]
	if len(tokens) == 0 {
		return nil, valuePosition.Errorf("empty OID definition")
	}
	var oid asn1go.OID
	firstToken := tokens[0]
	switch firstToken.Type() {
	case IDENT:
		defintion, ok := d.definitions[tokens[0].String()]
		if !ok {
			return nil, valuePosition.Errorf("unknown definition %s", tokens[0].String())
		}
		oidDefintion, ok := defintion.(OIDValue)
		if !ok {
			return nil, valuePosition.Errorf("definition %s is not an OID", tokens[0].String())
		}
		oid = oidDefintion.OID()
	case NUMBER:
		oid, err = asn1go.ParseOID(firstToken.String(), d.OIDLookup)
		if err != nil {
			return nil, err
		}
	default:
		return nil, valuePosition.Errorf("unexpected token %s", tokens[0].String())
	}
	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		switch token.Type() {
		case NUMBER:
			tail, err := asn1go.ParseOID(token.String(), d.OIDLookup)
			if err != nil {
				return nil, err
			}
			oid = append(oid, tail...)
		case IDENT:
			if len(tokens) > i+3 && tokens[i+1].TokenIs("(") && tokens[i+3].TokenIs(")") {
				n, err := strconv.Atoi(tokens[i+2].String())
				if err != nil {
					return nil, tokens[i+2].WrapError(err)
				}
				oid = append(oid, n)
				i += 3
			} else {
				ref, ok := d.definitions[token.String()]
				if !ok {
					return nil, token.source.Errorf("unknown reference %s", token.String())
				}
				oidDefintion, ok := ref.(OIDValue)
				if !ok {
					return nil, token.source.Errorf("reference %s is not an OID", token.String())
				}
				oid = nil
				oid = append(oid, oidDefintion.OID()...)
			}
		default:
			return nil, valuePosition.Errorf("unexpected token %s", token.String())
		}
	}

	return &oidDefintion{oid: oid, source: firstToken.source}, nil
}

func (reader *oidReader) Source() Position {
	return builtInPosition
}
