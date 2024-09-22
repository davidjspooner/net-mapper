package asn1mib

import (
	"context"
	"strconv"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1go"
)

// -----------------------
type OIDValue interface {
	OID(ctx context.Context) (asn1go.OID, error)
}
type oidDefintion struct {
	oid    asn1go.OID
	tokens TokenList
	source Position
}

func (o *oidDefintion) OID(ctx context.Context) (asn1go.OID, error) {
	if len(o.oid) == 0 {
		var oid asn1go.OID
		//decode the oid from the tokens
		for i := 0; i < o.tokens.Length(); i++ {
			token, _ := o.tokens.LookAhead(i)
			switch token.Type() {
			case NUMBER:
				tail, err := asn1go.ParseOID(token.String(), func(s string) (asn1go.OID, error) {
					oidDef, err := Lookup[OIDValue](ctx, s)
					if err != nil {
						return nil, token.WrapError(err)
					}
					return oidDef.OID(ctx)
				})
				if err != nil {
					return nil, err
				}
				oid = append(oid, tail...)
			case IDENT:

				peek1, _ := o.tokens.LookAhead(i + 1)
				peek2, _ := o.tokens.LookAhead(i + 2)
				peek3, _ := o.tokens.LookAhead(i + 3)

				if o.tokens.Length() > i+3 && peek1.IsText("(") && peek3.IsText(")") {
					n, err := strconv.Atoi(peek2.String())
					if err != nil {
						return nil, peek2.WrapError(err)
					}
					oid = append(oid, n)
					i += 3
				} else {
					oidDefintion, err := Lookup[OIDValue](ctx, token.String())
					if err != nil {
						return nil, token.WrapError(err)
					}
					oid = nil
					oidOther, err := oidDefintion.OID(ctx)
					if err != nil {
						return nil, token.WrapError(err)
					}
					oid = append(oid, oidOther...)
				}
			default:
				return nil, o.source.Errorf("unexpected token %s", token.String())
			}
		}
		o.oid = oid
	}
	return o.oid, nil
}
func (o *oidDefintion) Source() Position {
	return o.source
}
