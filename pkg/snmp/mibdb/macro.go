package mibdb

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type MacroDefintion struct {
	valueBase
	name   string
	fields map[string]Type
}

var _ Type = (*MacroDefintion)(nil)

func (mibMacro *MacroDefintion) readDefinition(ctx context.Context, module *Module, s mibtoken.Reader) error {
	block, err := mibtoken.ReadBlock(s, "BEGIN", "END")
	if err != nil {
		return err
	}
	for !block.IsEOF() {
		token, err := block.Pop()
		if err != nil {
			return err
		}
		ttype := token.Type()
		if ttype != mibtoken.IDENT {
			return token.WrapError(asn1error.NewUnexpectedError("UPPERCASETOKEN", token.String(), "macro element"))
		}
		err = block.ReadExpected("::=")
		if err != nil {
			return err
		}
		choices := &patternChoice{}
		choices.set(module, block, *token.Source())
		err = choices.readDefinition(ctx, module, block)
		fieldName := token.String()

		if err != nil {
			return err
		}
		if mibMacro.fields == nil {
			mibMacro.fields = make(map[string]Type)
		}
		if _, ok := mibMacro.fields[fieldName]; ok {
			return token.WrapError(asn1error.NewUnexpectedError("DUPLICATE", fieldName, "macro field"))
		}
		switch len(choices.alternatives) {
		case 0:
			return token.WrapError(asn1error.NewUnexpectedError("EMPTY", fieldName, "macro field"))
		case 1:
			mibMacro.fields[fieldName] = choices.alternatives[0]
		case 2:
			secondChoice := choices.alternatives[1]
			if seq, ok := secondChoice.(*patternSequence); ok {
				if len(seq.pattern) == 3 {
					firstInSeq, okFirst := seq.pattern[0].(*TypeReference)
					secondInSeq, okSecond := seq.pattern[1].(*ExpectedToken)
					thirdInSeq, okThird := seq.pattern[2].(*TypeReference)
					_ = secondInSeq
					if okFirst && okSecond && okThird {
						if firstInSeq.ident.String() == fieldName {
							seqOf := &TypeReference{
								ident:      thirdInSeq.ident,
								valueBase:  choices.valueBase,
								sequenceOf: true,
							}
							mibMacro.fields[fieldName] = seqOf
							continue
						} else if thirdInSeq.ident.String() == fieldName {
							seqOf := &TypeReference{
								ident:      firstInSeq.ident,
								valueBase:  choices.valueBase,
								sequenceOf: true,
							}
							mibMacro.fields[fieldName] = seqOf
							continue
						}
					}
				}
			}
			mibMacro.fields[fieldName] = choices
		default:
			mibMacro.fields[fieldName] = choices
		}
	}

	return nil
}

func (mibMacro *MacroDefintion) Source() mibtoken.Source {
	return mibMacro.source
}

func (mibMacro *MacroDefintion) Name() string {
	return mibMacro.name
}

func (mibMacro *MacroDefintion) String() string {
	return mibMacro.Name()
}

func (mibMacro *MacroDefintion) readValue(ctx context.Context, module *Module, s mibtoken.Reader) (Value, error) {

	ctx = withLookupContext(ctx, func(ctx context.Context, name string) (Definition, *Module, error) {
		return mibMacro.fields[name], nil, nil
	})

	typeNotation := mibMacro.fields["TYPE NOTATION"]
	if typeNotation == nil {
		return nil, fmt.Errorf("missing TYPE NOTATION in macro %s", mibMacro.Name())
	}

	value, err := typeNotation.readValue(ctx, module, s)
	if err != nil {
		return nil, err
	}
	return value, nil

}
