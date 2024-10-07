package mibdb

import (
	"context"
	"slices"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type ValueReader interface {
	readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error)
}

type Type interface {
	Definition
	ValueReader
	readDefinition(ctx context.Context, module *Module, s mibtoken.Reader) error
}

type TypeReference struct {
	valueBase
	ident      *mibtoken.Token
	sequenceOf bool
	constraint *mibtoken.List
}

var _ Type = (*TypeReference)(nil)

var simpleTypeNames = []string{"INTEGER", "OCTET STRING", "SEQUENCE", "SEQUENCE OF", "CHOICE", "OBJECT IDENTIFIER", "IA5String"}

var brackets = map[string]string{"{": "}", "(": ")", "[": "]"}

func (ref *TypeReference) readDefinition(_ context.Context, module *Module, s mibtoken.Reader) error {
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
		ref.ident = peek
		s.Pop()

		if ref.ident.String() == "SEQUENCE OF" {
			ref.sequenceOf = true
			ref.ident, _ = s.Pop()
		}

		//check later in "compile" if the ident is a known type

		peek, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		closer, ok := brackets[peek.String()]
		if !ok {
			return nil
		}
		opener := peek.String()
		ref.constraint, err = mibtoken.ReadBlock(s, opener, closer)
		if err != nil {
			return err
		}
		return nil
	default:
		return peek.WrapError(asn1error.NewUnexpectedError("IDENT", peek.String(), "token"))
	}
}

func (ref *TypeReference) Source() mibtoken.Source {
	return ref.source
}

func (ref *TypeReference) compileValue(ctx context.Context, module *Module) error {
	err := ref.valueBase.compileMeta(ctx)
	if err != nil {
		return err
	}
	ok := slices.Contains(simpleTypeNames, ref.ident.String())
	if !ok {
		return ref.ident.WrapError(asn1error.NewUnexpectedError("KNOWNTYPE", ref.ident.String(), "SimpleType.readDefinition"))
	}
	return nil
}

func (ref *TypeReference) readOneValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {
	valueType := ref.ident.String()
	switch valueType {
	case "OBJECT IDENTIFIER":
		return ref.readObjectIdentifierValue(ctx, s)
	case "IA5String":
		tok, err := s.Pop()
		if err != nil {
			return nil, err
		}
		if tok.Type() != mibtoken.STRING {
			return nil, tok.WrapError(asn1error.NewUnexpectedError("\"STRING\"", tok.String(), "SimpleType.readValue"))
		}
		value := &goValue[string]{value: tok.String()}
		value.set(module, ref.metaTokens, *tok.Source())
		return value, nil
	case "value":
		//TODO read an identifier defintion....
		other := &ConstantValue{}
		other.set(module, ref.metaTokens, *s.Source())
		err := other.read(ctx, s)
		if err != nil {
			return nil, err
		}
		return other, nil
	case "identifier":
		//TODO read an identifier defintion....
		other := &TypeReference{}
		other.set(module, ref.metaTokens, *s.Source())
		err := other.readDefinition(ctx, module, s)
		if err != nil {
			return nil, err
		}
		return other, nil
	case "type":
		other := &TypeReference{}
		other.set(module, ref.metaTokens, *s.Source())
		err := other.readDefinition(ctx, module, s)
		if err != nil {
			return nil, err
		}
		return other, nil
	case "empty":
		return nil, nil
	default:
		def, _, err := Lookup[Type](ctx, valueType)
		if err != nil {
			return nil, s.Source().WrapError(err)
		}
		value, err := def.readValue(ctx, module, s)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func (ref *TypeReference) readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {

	depth := getDepth(ctx)
	if depth.Inc(ref) > 100 {
		return nil, ref.source.Errorf("depth limit reached")
	}
	defer depth.Dec(ref)
	if !ref.sequenceOf {
		value, err := ref.readOneValue(ctx, module, s)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
	values := ValueList{}
	peek, err := s.LookAhead(0)
	if err != nil {
		return nil, err
	}
	if peek.String() != "}" {
	loop:
		for !s.IsEOF() {
			value, err := ref.readOneValue(ctx, module, s)
			if err != nil {
				return nil, err
			}
			if value != nil {
				values = append(values, value)
			}
			peek, err := s.LookAhead(0)
			if err != nil {
				return nil, err
			}
			switch peek.String() {
			case ",":
				s.Pop()
				continue
			case "}":
				break loop
			default:
				return nil, peek.WrapError(asn1error.NewUnexpectedError("',' or '}'", peek.String(), "token"))
			}
		}
	}
	return &values, nil
}

func (ref *TypeReference) readObjectIdentifierValue(ctx context.Context, s mibtoken.Reader) (Value, error) {
	oidValue := &OidValue{}
	err := oidValue.readOid(ctx, s)
	if err != nil {
		return nil, err
	}
	return oidValue, nil
}

// ------------------------------------
