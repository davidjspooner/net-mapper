package mibdb

import (
	"context"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type fieldDefintion struct {
	fieldName *mibtoken.Token
	tokens    mibtoken.List
}

func (fieldDef *fieldDefintion) read(ctx context.Context, s mibtoken.Queue) error {
	for !s.IsEOF() {
		peek1, err := s.LookAhead(1)
		if err == nil && peek1.String() == "::=" {
			return nil
		}
		tok, _ := s.Pop()
		fieldDef.tokens.AppendTokens(tok)
	}
	return nil
}

type MacroDefintion struct {
	source     mibtoken.Position
	metaTokens *mibtoken.List
	fields     map[string]*fieldDefintion
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
		if ttype != mibtoken.KEYWORD && ttype != mibtoken.IDENT {
			return token.WrapError(asn1core.NewUnexpectedError("UPPERCASETOKEN", token.String(), "macro element"))
		}
		err = block.PopExpected("::=")
		if err != nil {
			return err
		}
		elementDef := &fieldDefintion{fieldName: token}
		err = elementDef.read(ctx, block)
		if err != nil {
			return err
		}
		if mibMacro.fields == nil {
			mibMacro.fields = make(map[string]*fieldDefintion)
		}
		mibMacro.fields[elementDef.fieldName.String()] = elementDef
	}

	return nil
}

func (mibMacro *MacroDefintion) Source() mibtoken.Position {
	return mibMacro.source
}

type FieldValue struct {
	fieldName *mibtoken.Token
	mibValue  *Oid
}

type MacroInvocation struct {
	source     mibtoken.Position
	use        *MacroDefintion
	metaTokens *mibtoken.List
	fields     []FieldValue
}

func (mibMacroInvocation *MacroInvocation) read(ctx context.Context, s mibtoken.Queue) error {
	return mibMacroInvocation.source.WrapError(asn1core.NewUnimplementedError("MacroInvocation.read").MaybeLater())
}

func (mibMacroInvocation *MacroInvocation) Source() mibtoken.Position {
	return mibMacroInvocation.source
}
