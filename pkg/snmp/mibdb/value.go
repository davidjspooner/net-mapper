package mibdb

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type valueBase struct {
	module     *Module
	metaTokens *mibtoken.List
	source     mibtoken.Source
}

func (base *valueBase) set(module *Module, metaTokens *mibtoken.List, source mibtoken.Source) {
	base.module = module
	base.metaTokens = metaTokens
	base.source = source
}

func (base *valueBase) compileMeta(ctx context.Context) error {
	return nil
	if base.metaTokens == nil || base.metaTokens.Length() == 0 {
		return nil
	}
	copy := base.metaTokens.Clone()
	tok, err := copy.Pop()
	if err != nil {
		return err
	}
	def, _, err := Lookup[Definition](ctx, tok.String())
	if err != nil {
		return nil
	}
	_, ok := def.(*SimpleType)
	if ok {
		return nil
	}
	macro, ok := def.(*MacroDefintion)
	if ok {
		value, err := macro.readValue(ctx, base.module, copy)
		if err != nil {
			return err
		}
		_ = value
	}
	return nil
}

// ------------------------------------

type Value interface {
	Definition
	compileValue(ctx context.Context, module *Module) error
}

// ------------------------------------

type OidValue struct {
	valueBase
	elements []string
	compiled asn1go.OID
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
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return err
	}
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
			otherDefintion, otherModule, err := Lookup[Definition](ctx, element)
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
	valueBase
	elements []string
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

// ------------------------------------

type structureValue struct {
	valueBase
	vType  Type
	fields map[string]Value
}

func (value *structureValue) Source() mibtoken.Source {
	return value.source
}

func (value *structureValue) compileValue(ctx context.Context, module *Module) error {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return err
	}
	return nil
}

// ------------------------------------

type goValue[T any] struct {
	valueBase
	value T
}

var _ Value = (*goValue[string])(nil)

func (value *goValue[T]) Source() mibtoken.Source {
	return value.source
}

func (value *goValue[T]) compileValue(ctx context.Context, module *Module) error {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return err
	}
	return nil
}
