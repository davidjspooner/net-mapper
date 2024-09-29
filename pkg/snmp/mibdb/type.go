package mibdb

import (
	"context"
	"slices"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Type interface {
	Definition
	compile(ctx context.Context) error
	readDefinition(ctx context.Context, s mibtoken.Queue) error
	readValue(ctx context.Context, s mibtoken.Queue) (Value, error)
}

type SimpleType struct {
	source     mibtoken.Position
	ident      *mibtoken.Token
	constraint *mibtoken.List
	metaTokens *mibtoken.List
}

var _ Type = (*SimpleType)(nil)

var simpleTypeNames = []string{"INTEGER", "OCTET STRING", "SEQUENCE", "SEQUENCE OF", "CHOICE", "OBJECT IDENTIFIER", "IA5String"}

var brackets = map[string]string{"{": "}", "(": ")", "[": "]"}

func (simpleType *SimpleType) readDefinition(_ context.Context, s mibtoken.Queue) error {
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
	case mibtoken.IDENT:
		simpleType.ident = peek

		ok := slices.Contains(simpleTypeNames, peek.String())
		if !ok {
			return peek.WrapError(asn1core.NewUnexpectedError("KNOWNTYPE", peek.String(), "SimpleType.readDefinition"))
		}

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
		simpleType.constraint, err = s.PopBlock(opener, closer)
		if err != nil {
			return err
		}
		return nil
	default:
		return peek.WrapError(asn1core.NewUnexpectedError("IDENT", peek.String(), "token"))
	}
}

func (simpleType *SimpleType) Source() mibtoken.Position {
	return simpleType.source
}

func (simpleType *SimpleType) compile(ctx context.Context) error {
	return nil
}

func (simpleType *SimpleType) readValue(ctx context.Context, s mibtoken.Queue) (Value, error) {
	valueType := simpleType.ident.String()
	switch valueType {
	case "OBJECT IDENTIFIER":
		return simpleType.readObjectIdentifierValue(ctx, s)
	}

	return nil, asn1core.NewUnimplementedError("simpleType.readValue of type %s", valueType).MaybeLater()
}

func (simpleType *SimpleType) readObjectIdentifierValue(ctx context.Context, s mibtoken.Queue) (Value, error) {
	oidValue := &OidValue{}
	err := oidValue.readOid(ctx, s)
	if err != nil {
		return nil, err
	}
	return oidValue, nil
}

type structureType struct {
	source mibtoken.Position
	tokens mibtoken.List
}

var _ Type = (*structureType)(nil)

func (structureType *structureType) Source() mibtoken.Position {
	return structureType.source
}

func (structureType *structureType) compile(ctx context.Context) error {
	return nil
}

func (structureType *structureType) readDefinition(ctx context.Context, s mibtoken.Queue) error {
	for !s.IsEOF() {
		peek1, err := s.LookAhead(1)
		if err == nil && peek1.String() == "::=" {
			return nil
		}
		//read one peek set
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		switch peek.Type() {
		case mibtoken.IDENT:
			structureType.tokens.AppendTokens(peek)
			s.Pop()
			peek, err := s.LookAhead(0)
			if err == nil {
				close, ok := brackets[peek.String()]
				if ok {
					open := peek.String()
					constraint, err := s.PopBlock(open, close)
					if err != nil {
						return err
					}
					structureType.tokens.AppendTokens(peek)
					structureType.tokens.AppendLists(constraint)
					structureType.tokens.AppendTokens(mibtoken.New(close, *peek.Source()))
				}
			}
			//todo the token plus brackets
		case mibtoken.STRING:
			structureType.tokens.AppendTokens(peek)
			s.Pop()
		case mibtoken.SYMBOL:
			peekTxt := peek.String()
			switch peekTxt {
			case "|":
				return nil
			default:
				return peek.WrapError(asn1core.NewUnexpectedError("IDENT", peek.String(), "token in structureType.read"))
			}
		}
	}
	return nil
}

func (structureType *structureType) isDefined() bool {
	return structureType.tokens.Length() > 0
}

func (structureType *structureType) String() string {
	return structureType.tokens.String()
}

func (structureType *structureType) readValue(ctx context.Context, s mibtoken.Queue) (Value, error) {
	return nil, asn1core.NewUnimplementedError("structureType.readValue").MaybeLater()
}

type choiceType struct {
	source       mibtoken.Position
	alternatives []*structureType
}

var _ Type = (*choiceType)(nil)

func (choiceType *choiceType) readDefinition(ctx context.Context, s mibtoken.Queue) error {
	alternative := &structureType{}
	for !s.IsEOF() {
		alternative = &structureType{}
		if len(choiceType.alternatives) > 0 {
			peek, _ := s.LookAhead(0)
			if peek.String() != "|" {
				break
			}
			s.Pop()
		}
		err := alternative.readDefinition(ctx, s)
		if err != nil {
			return err
		}
		if !alternative.isDefined() {
			break
		}
		choiceType.alternatives = append(choiceType.alternatives, alternative)
	}
	if alternative.isDefined() {
		choiceType.alternatives = append(choiceType.alternatives, alternative)
	}
	return nil
}

func (choiceType *choiceType) Source() mibtoken.Position {
	return choiceType.source
}

func (choiceType *choiceType) compile(ctx context.Context) error {
	return nil
}

func (choiceType *choiceType) readValue(ctx context.Context, s mibtoken.Queue) (Value, error) {
	return nil, asn1core.NewUnimplementedError("choiceType.readValue").MaybeLater()
}
