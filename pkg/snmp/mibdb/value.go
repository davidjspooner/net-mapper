package mibdb

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Value interface {
	Definition
	compileValue(ctx context.Context, module *Module) error
}

// ------------------------------------

type OidValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Source
	compiled   asn1go.OID
}

var _ Definition = (*OidValue)(nil)

func (value *OidValue) readOid(_ context.Context, s mibtoken.Reader) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := mibtoken.ReadBlock(s, "{", "}")
		if err != nil {
			return err
		}
		for !elements.IsEOF() {
			element, err := elements.Pop()
			if err != nil {
				return err
			}
			peek, err := elements.LookAhead(0)
			if err == nil && peek.String() == "(" {
				block, err := mibtoken.ReadBlock(elements, "(", ")")
				if err != nil {
					return err
				}
				element, err = block.Pop()
				if err != nil {
					return err
				}

			}
			value.elements = append(value.elements, element.String())
		}
		return nil
	}
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *OidValue) Source() mibtoken.Source {
	return value.source
}

func (value *OidValue) compileValue(ctx context.Context, module *Module) error {
	if len(value.compiled) > 0 {
		return nil
	}
	for _, element := range value.elements {

		n, err := strconv.Atoi(element)
		if err == nil {
			value.compiled = append(value.compiled, n)
			continue
		}
		value.compiled = nil
		switch element {
		case "iso":
			value.compiled = append(value.compiled, 1)
		default:
			otherDefintion, otherModule, err := LookupInModule[Definition](ctx, module, element)
			if err != nil {
				return value.source.WrapError(err)
			}
			otherOID, ok := otherDefintion.(*OidValue)
			if !ok {
				return value.source.Errorf("expected OID but got %T", otherDefintion)
			}
			err = otherOID.compileValue(ctx, otherModule)
			if err != nil {
				return value.source.WrapError(err)
			}
			value.compiled = append(value.compiled, otherOID.compiled...)
		}
	}
	return nil
}

func (value *OidValue) String() string {
	if len(value.compiled) == 0 {
		return "<uncompiled>"
	}
	return value.compiled.String()
}

// ------------------------------------

type ConstantValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Source
}

var _ Definition = (*ConstantValue)(nil)

func (value *ConstantValue) read(_ context.Context, s mibtoken.Reader) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := mibtoken.ReadBlock(s, "{", "}")
		if err != nil {
			return err
		}
		for !elements.IsEOF() {
			element, err := elements.Pop()
			if err != nil {
				return err
			}
			value.elements = append(value.elements, element.String())
		}
		return nil
	}
	s.Pop()
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *ConstantValue) Source() mibtoken.Source {
	return value.source
}

func (value *ConstantValue) compile(_ context.Context) error {
	return nil
}

// ------------------------------------

type structureValue struct {
	source     mibtoken.Source
	vType      Type
	metaTokens *mibtoken.List
	fields     map[string]Value
}

func (value *structureValue) Source() mibtoken.Source {
	return value.source
}

func (value *structureValue) compileValue(ctx context.Context, module *Module) error {
	return nil
}

// ------------------------------------

type goValue[T any] struct {
	value  T
	source mibtoken.Source
}

var _ Value = (*goValue[string])(nil)

func (value *goValue[T]) Source() mibtoken.Source {
	return value.source
}

func (value *goValue[T]) compileValue(ctx context.Context, module *Module) error {
	return nil
}
