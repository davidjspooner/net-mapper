package mibdb

import (
	"context"
	"slices"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Type interface {
	Definition
	readDefinition(ctx context.Context, s mibtoken.Reader) error
	readValue(ctx context.Context, s mibtoken.Reader) (Value, error)
}

type SimpleType struct {
	source     mibtoken.Source
	ident      *mibtoken.Token
	constraint *mibtoken.List
	metaTokens *mibtoken.List
}

var _ Type = (*SimpleType)(nil)

var simpleTypeNames = []string{"INTEGER", "OCTET STRING", "SEQUENCE", "SEQUENCE OF", "CHOICE", "OBJECT IDENTIFIER", "IA5String"}

var brackets = map[string]string{"{": "}", "(": ")", "[": "]"}

func (simpleType *SimpleType) readDefinition(_ context.Context, s mibtoken.Reader) error {
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}

	if peek.String() == "[" {
		envelope, err := mibtoken.ReadBlock(s, "[", "]")
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

		//check later in "compile" if the ident is a known type

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
		simpleType.constraint, err = mibtoken.ReadBlock(s, opener, closer)
		if err != nil {
			return err
		}
		return nil
	default:
		return peek.WrapError(asn1core.NewUnexpectedError("IDENT", peek.String(), "token"))
	}
}

func (simpleType *SimpleType) Source() mibtoken.Source {
	return simpleType.source
}

func (simpleType *SimpleType) compileValue(ctx context.Context, module *Module) error {
	ok := slices.Contains(simpleTypeNames, simpleType.ident.String())
	if !ok {
		return simpleType.ident.WrapError(asn1core.NewUnexpectedError("KNOWNTYPE", simpleType.ident.String(), "SimpleType.readDefinition"))
	}
	return nil
}

func (simpleType *SimpleType) readValue(ctx context.Context, s mibtoken.Reader) (Value, error) {
	valueType := simpleType.ident.String()
	switch valueType {
	case "OBJECT IDENTIFIER":
		return simpleType.readObjectIdentifierValue(ctx, s)
	case "IA5String":
		tok, err := s.Pop()
		if err != nil {
			return nil, err
		}
		if tok.Type() != mibtoken.STRING {
			return nil, tok.WrapError(asn1core.NewUnexpectedError("\"STRING\"", tok.String(), "SimpleType.readValue"))
		}
		return &goValue[string]{value: tok.String(), source: *tok.Source()}, nil
	case "type":
		//TODO read a type defintion....
		other := &SimpleType{}
		err := other.readDefinition(ctx, s)
		if err != nil {
			return nil, err
		}
		return other, nil
	}

	return nil, asn1core.NewUnimplementedError("simpleType.readValue of type %s", valueType).TODO()
}

func (simpleType *SimpleType) readObjectIdentifierValue(ctx context.Context, s mibtoken.Reader) (Value, error) {
	oidValue := &OidValue{}
	err := oidValue.readOid(ctx, s)
	if err != nil {
		return nil, err
	}
	return oidValue, nil
}

type sequenceType struct {
	source mibtoken.Source
	tokens mibtoken.List
}

var _ Type = (*sequenceType)(nil)

func (sequenceType *sequenceType) Source() mibtoken.Source {
	return sequenceType.source
}

func (sequenceType *sequenceType) readDefinition(ctx context.Context, s mibtoken.Reader) error {
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
			sequenceType.tokens.AppendTokens(peek)
			s.Pop()
			peek, err := s.LookAhead(0)
			if err == nil {
				close, ok := brackets[peek.String()]
				if ok {
					open := peek.String()
					constraint, err := mibtoken.ReadBlock(s, open, close)
					if err != nil {
						return err
					}
					sequenceType.tokens.AppendTokens(peek)
					sequenceType.tokens.AppendLists(constraint)
					sequenceType.tokens.AppendTokens(mibtoken.New(close, *peek.Source()))
				}
			}
			//todo the token plus brackets
		case mibtoken.STRING:
			sequenceType.tokens.AppendTokens(peek)
			s.Pop()
		case mibtoken.SYMBOL:
			peekTxt := peek.String()
			switch peekTxt {
			case "|":
				return nil
			default:
				return peek.WrapError(asn1core.NewUnexpectedError("IDENT", peek.String(), "token in sequenceType.read"))
			}
		}
	}
	return nil
}

func (sequenceType *sequenceType) isDefined() bool {
	return sequenceType.tokens.Length() > 0
}

func (sequenceType *sequenceType) String() string {
	return sequenceType.tokens.String()
}

func (sequenceType *sequenceType) readValue(ctx context.Context, s mibtoken.Reader) (Value, error) {

	overall := &structureValue{}

	err := sequenceType.tokens.ForEach(func(definition *mibtoken.Token) error {
		switch definition.Type() {
		case mibtoken.IDENT:
			defStr := definition.String()
			if defStr == "empty" {
				return nil
			}
			typeReader, err := Lookup[Type](ctx, defStr)
			if err != nil {
				return mibtoken.WrapError(s, err)
			}
			value, err := typeReader.readValue(ctx, s)
			if err != nil {
				return mibtoken.WrapError(s, err)
			}
			_ = value
		case mibtoken.STRING:
			expected, err := mibtoken.Unquote(definition)
			if err != nil {
				return mibtoken.WrapError(s, err)
			}
			err = mibtoken.ReadExpected(s, expected)
			if err != nil {
				return mibtoken.WrapError(s, err)
			}
		default:
			return mibtoken.WrapError(s, asn1core.NewUnexpectedError("IDENT", definition.String(), "token in sequenceType.readValue"))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return overall, nil
}

type choiceType struct {
	source       mibtoken.Source
	alternatives []Type
}

var _ Type = (*choiceType)(nil)

func (choiceType *choiceType) readDefinition(ctx context.Context, s mibtoken.Reader) error {
	for !s.IsEOF() {
		if len(choiceType.alternatives) > 0 {
			peek, _ := s.LookAhead(0)
			if peek.String() != "|" {
				break
			}
			s.Pop()
		}
		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		switch peek.String() {
		case "value":
			mibtoken.ReadExpected(s, "value")
			block, err := mibtoken.ReadBlock(s, "(", ")")
			if err != nil {
				return err
			}
			label, err := block.Pop()
			if err != nil {
				return err
			}
			typeName, err := block.Pop()
			if err != nil {
				typeName = label
			}
			simpleType := &SimpleType{ident: typeName, source: *typeName.Source()}
			choiceType.alternatives = append(choiceType.alternatives, simpleType)
		case "type":
			mibtoken.ReadExpected(s, "type")
			block, err := mibtoken.ReadBlock(s, "(", ")")
			if err != nil {
				simpleType := &SimpleType{ident: peek, source: *peek.Source()}
				choiceType.alternatives = append(choiceType.alternatives, simpleType)
				continue
			}
			typeName, err := block.Pop()
			if err != nil {
				return err
			}
			typeDef, err := Lookup[Type](ctx, typeName.String())
			if err != nil {
				return err
			}
			choiceType.alternatives = append(choiceType.alternatives, typeDef)
		default:
			alternative := &sequenceType{}
			err := alternative.readDefinition(ctx, s)
			if err != nil {
				return err
			}
			if !alternative.isDefined() {
				break
			}
			choiceType.alternatives = append(choiceType.alternatives, alternative)
		}
	}
	return nil
}

func (choiceType *choiceType) Source() mibtoken.Source {
	return choiceType.source
}

func (choiceType *choiceType) readValue(ctx context.Context, s mibtoken.Reader) (Value, error) {

	errs := asn1core.ErrorList{}
	for _, alt := range choiceType.alternatives {
		tmp := mibtoken.NewProjection(s)
		value, err := alt.readValue(ctx, tmp)
		if err == nil {
			tmp.Commit()
			return value, nil
		}
		errs = append(errs, err)
	}
	switch len(errs) {
	case 1:
		return nil, errs[0]
	case 0:
		return nil, s.Source().Errorf("no choice matched")
	default:
		return nil, errs
	}
}
