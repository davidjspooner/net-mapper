package mibdb

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type valueBase struct {
	module     *Module
	metaTokens *mibtoken.List
	metaValue  Value
	source     mibtoken.Source
}

func (base *valueBase) set(module *Module, metaTokens *mibtoken.List, source mibtoken.Source) {
	base.module = module
	base.metaTokens = metaTokens
	base.source = source
}

func (base *valueBase) compileMeta(ctx context.Context) error {
	if base.metaTokens == nil || base.metaTokens.Length() == 0 || base.metaValue != nil {
		return nil
	}

	copy := mibtoken.NewProjection(base.metaTokens)
	tok, err := copy.Pop()
	if err != nil {
		return err
	}
	def, _, err := Lookup[Definition](ctx, tok.String())
	if err != nil {
		tokStr := tok.String()
		Lookup[Definition](ctx, tokStr)
		return tok.WrapError(err)
	}
	_, ok := def.(*TypeReference)
	if ok {
		return nil
	}
	macro, ok := def.(*MacroDefintion)
	if ok {
		value, err := macro.readValue(ctx, base.module, copy)
		if err != nil {
			return err
		}
		base.metaValue = value
	}
	return nil
}

// ------------------------------------

type Value interface {
	Definition
}

type CompilableValue interface {
	Value
	compileValue(ctx context.Context, module *Module) (Value, error)
}

// ------------------------------------

type OidValue struct {
	valueBase
	elements []string
	compiled asn1go.OID
}

var _ CompilableValue = (*OidValue)(nil)

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
	s.Pop()
	return nil
}

func (value *OidValue) Source() mibtoken.Source {
	return value.source
}

func (value *OidValue) compileValue(ctx context.Context, module *Module) (Value, error) {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return nil, err
	}
	if len(value.compiled) > 0 {
		return value, nil
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
				return nil, value.source.WrapError(err)
			}
			otherOID, ok := otherDefintion.(*OidValue)
			if !ok {
				return nil, value.source.Errorf("expected OID but got %T", otherDefintion)
			}
			ov, err := otherOID.compileValue(ctx, otherModule)
			if err != nil {
				return nil, value.source.WrapError(err)
			}
			otherOID = ov.(*OidValue)
			value.compiled = append(value.compiled, otherOID.compiled...)
		}
	}
	return value, nil
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

var _ CompilableValue = (*ConstantValue)(nil)

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

func (value *ConstantValue) compileValue(ctx context.Context, module *Module) (Value, error) {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (value *ConstantValue) Source() mibtoken.Source {
	return value.source
}

// ------------------------------------

type ExpectedToken struct {
	text string
}

func (expected *ExpectedToken) readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {
	peek, err := s.LookAhead(0)
	if err != nil {
		return nil, err
	}
	actual := peek.String()
	if actual == expected.text {
		s.Pop()
		return nil, nil
	}
	if actual == "ACCESS" && expected.text == "MAX-ACCESS" {
		s.Pop()
		return nil, nil
	}
	return nil, peek.WrapError(asn1error.NewUnexpectedError(expected.text, actual, "mib token"))
}

// ------------------------------------

type CompositeValue struct {
	valueBase
	vType  Type
	fields map[string]Value
}

var _ CompilableValue = (*CompositeValue)(nil)

func (value *CompositeValue) Source() mibtoken.Source {
	return value.source
}

func (value *CompositeValue) compileValue(ctx context.Context, module *Module) (Value, error) {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ------------------------------------

type goValue[T any] struct {
	valueBase
	value T
}

var _ CompilableValue = (*goValue[string])(nil)

func (value *goValue[T]) Source() mibtoken.Source {
	return value.source
}

func (value *goValue[T]) compileValue(ctx context.Context, module *Module) (Value, error) {
	err := value.valueBase.compileMeta(ctx)
	if err != nil {
		return nil, err
	}
	return value, nil
}

//--------------------------------------

type ValueList []Value

var _ CompilableValue = (ValueList)(nil)

func (list ValueList) Source() mibtoken.Source {
	if len(list) == 0 {
		return mibtoken.Source{}
	}
	return list[0].Source()
}

func (list ValueList) compileValue(ctx context.Context, module *Module) (Value, error) {
	for i, value := range list {
		compilable, ok := value.(CompilableValue)
		if ok {
			value, err := compilable.compileValue(ctx, module)
			if err != nil {
				return nil, err
			}
			list[i] = value
		}
	}
	return list, nil
}
