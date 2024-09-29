package mibdb

import (
	"context"
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type MacroDefintion struct {
	source     mibtoken.Position
	metaTokens *mibtoken.List
	fields     map[string]Type
}

var _ Definition = (*MacroDefintion)(nil)

func (mibMacro *MacroDefintion) read(ctx context.Context, s mibtoken.Queue) error {
	block, err := s.PopBlock("BEGIN", "END")
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
		err = block.PopExpected("::=")
		if err != nil {
			return err
		}
		choices := &choiceType{source: *block.Source()}
		err = choices.readDefinition(ctx, block)
		fieldName := token.String()

		fmt.Printf("DEBUG   Field: %s\n", fieldName)
		for _, alt := range choices.alternatives {
			fmt.Printf("DEBUG      Alternative: %s\n", alt.String())
		}

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

func (mibMacro *MacroDefintion) Source() mibtoken.Position {
	return mibMacro.source
}

func (mibMacro *MacroDefintion) compile(ctx context.Context) error {
	return asn1core.NewUnimplementedError("MacroDefintion.Compile").MaybeLater()
}
