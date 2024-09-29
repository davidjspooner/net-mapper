package mibdb

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type MacroDefintion struct {
	name       string
	source     mibtoken.Source
	metaTokens *mibtoken.List
	fields     map[string]Type
}

var _ Type = (*MacroDefintion)(nil)

func (mibMacro *MacroDefintion) readDefinition(ctx context.Context, s mibtoken.Reader) error {
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
			return token.WrapError(asn1core.NewUnexpectedError("UPPERCASETOKEN", token.String(), "macro element"))
		}
		err = block.ReadExpected("::=")
		if err != nil {
			return err
		}
		choices := &choiceType{source: *block.Source()}
		err = choices.readDefinition(ctx, block)
		fieldName := token.String()

		if err != nil {
			return err
		}
		if mibMacro.fields == nil {
			mibMacro.fields = make(map[string]Type)
		}
		if _, ok := mibMacro.fields[fieldName]; ok {
			return token.WrapError(asn1core.NewUnexpectedError("DUPLICATE", fieldName, "macro field"))
		}
		if len(choices.alternatives) == 1 {
			mibMacro.fields[fieldName] = choices.alternatives[0]
		} else {
			mibMacro.fields[fieldName] = choices
		}
	}

	return nil
}

func (mibMacro *MacroDefintion) Source() mibtoken.Source {
	return mibMacro.source
}

func (mibMacro *MacroDefintion) compile(ctx context.Context) error {
	return nil
}

func (mibMacro *MacroDefintion) Name() string {
	return mibMacro.name
}

func (mibMacro *MacroDefintion) String() string {
	return mibMacro.Name()
}

func (mibMacro *MacroDefintion) readValue(ctx context.Context, s mibtoken.Reader) (Value, error) {

	ctx = withContext(ctx, func(ctx context.Context, name string) (Definition, error) {
		return mibMacro.fields[name], nil
	})

	typeNotation := mibMacro.fields["TYPE NOTATION"]
	if typeNotation == nil {
		return nil, fmt.Errorf("missing TYPE NOTATION in macro %s", mibMacro.Name())
	}

	value, err := typeNotation.readValue(ctx, s)
	if err != nil {
		return nil, err
	}
	return value, nil

}
