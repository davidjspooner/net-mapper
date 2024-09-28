package mibdb

import (
	"context"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type SimpleType struct {
	source     mibtoken.Position
	ident      *mibtoken.Token
	criteria   *mibtoken.List
	metaTokens *mibtoken.List
}

var _ Definition = (*SimpleType)(nil)

var simpleTypeNames = []string{"INTEGER", "OCTET STRING", "SEQUENCE", "SEQUENCE OF", "CHOICE", "OBJECT IDENTIFIER", "IA5String"}

var brackets = map[string]string{"{": "}", "(": ")", "[": "]"}

func (mibSimpleType *SimpleType) read(ctx context.Context, s mibtoken.Queue) error {
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "[" {
		envelope, err := s.PopBlock("[", "]")
		if err != nil {
			return err
		}
		_ = envelope //TODO
		peek, err = s.LookAhead(0)
		if err != nil {
			return err
		}
		if peek.String() == "IMPLICIT" || peek.String() == "EXPLICIT" {
			s.Pop()
			peek, err = s.LookAhead(0)
			if err != nil {
				return err
			}
		}
	}

	switch peek.Type() {
	case mibtoken.IDENT, mibtoken.KEYWORD:
		mibSimpleType.ident = peek
		s.Pop()
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		closer, ok := brackets[peek.String()]
		if !ok {
			return nil
		}
		opener := peek.String()
		mibSimpleType.criteria, err = s.PopBlock(opener, closer)
		if err != nil {
			return err
		}
		return nil
	default:
		return peek.WrapError(asn1core.NewUnexpectedError("IDENT", peek.String(), "token"))
	}
}

func (mibSimpleType *SimpleType) Source() mibtoken.Position {
	return mibSimpleType.source
}
