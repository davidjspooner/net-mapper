package mibdb

import (
	"context"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Value interface {
	Definition
	compile(ctx context.Context) error
}

type OidValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Position
}

var _ Definition = (*OidValue)(nil)

func (value *OidValue) readOid(_ context.Context, s mibtoken.Queue) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := s.PopBlock("{", "}")
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
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *OidValue) Source() mibtoken.Position {
	return value.source
}

func (value *OidValue) compile(ctx context.Context) error {
	return asn1core.NewUnimplementedError("OidValue.Compile").MaybeLater()
}

type ConstantValue struct {
	elements   []string
	metaTokens *mibtoken.List
	source     mibtoken.Position
}

var _ Definition = (*ConstantValue)(nil)

func (value *ConstantValue) read(_ context.Context, s mibtoken.Queue) error {
	value.source = *s.Source()
	peek, err := s.LookAhead(0)
	if err != nil {
		return err
	}
	if peek.String() == "{" {
		elements, err := s.PopBlock("{", "}")
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
	value.elements = append(value.elements, peek.String())
	return nil
}

func (value *ConstantValue) Source() mibtoken.Position {
	return value.source
}

func (value *ConstantValue) compile(_ context.Context) error {
	return asn1core.NewUnimplementedError("ConstantValue.Compile").MaybeLater()
}

type structureValue struct {
	source     mibtoken.Position
	vType      Type
	metaTokens *mibtoken.List
	fields     map[string]Value
}

func (value *structureValue) Source() mibtoken.Position {
	return value.source
}

func (value *structureValue) read(_ context.Context, s mibtoken.Queue) error {
	//TODO use the macro definition to parse the invocation
	return value.source.WrapError(asn1core.NewUnimplementedError("structureValue.read").MaybeLater())
}

func (value *structureValue) compile(ctx context.Context) error {
	return asn1core.NewUnimplementedError("structureValue.Compile").MaybeLater()
}
