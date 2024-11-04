package mibdb

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

// ------------------------------------

type Object struct {
	valueBase
	name     string
	elements []string
	compiled asn1go.OID
}

var _ CompilableValue = (*Object)(nil)

func (object *Object) readOid(_ context.Context, s mibtoken.Reader) error {
	object.source = *s.Source()
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
			object.elements = append(object.elements, element.String())
		}
		return nil
	}
	object.elements = append(object.elements, peek.String())
	if object.name == "" {
		object.name = peek.String()
	}
	s.Pop()
	return nil
}

func (object *Object) Source() mibtoken.Source {
	return object.source
}

func (object *Object) Name() string {
	return object.name
}

func (object *Object) compileValue(ctx context.Context, module *Module) (Value, error) {
	err := object.valueBase.compileMeta(ctx)
	if err != nil {
		return nil, err
	}

	if len(object.compiled) > 0 {
		return object, nil
	}
	for _, element := range object.elements {

		n, err := strconv.Atoi(element)
		if err == nil {
			object.compiled = append(object.compiled, n)
			continue
		}
		object.compiled = nil
		switch element {
		case "iso":
			object.compiled = append(object.compiled, 1)
		default:
			otherDefintion, otherModule, err := Lookup[Definition](ctx, element)
			if err != nil {
				return nil, object.source.WrapError(err)
			}
			otherOID, ok := otherDefintion.(*Object)
			if !ok {
				return nil, object.source.Errorf("expected OID but got %T", otherDefintion)
			}
			ov, err := otherOID.compileValue(ctx, otherModule)
			if err != nil {
				return nil, object.source.WrapError(err)
			}
			otherOID = ov.(*Object)
			object.compiled = append(object.compiled, otherOID.compiled...)
		}
	}
	return object, nil
}

func (object *Object) OID() asn1go.OID {
	return object.compiled
}
