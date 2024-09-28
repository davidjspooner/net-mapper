package asn1binary

import (
	"fmt"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

type Envelope struct {
	Class asn1core.Class
	Tag   asn1core.Tag
}

func (e *Envelope) String() string {
	return fmt.Sprintf("[%s %s]", e.Class, e.Tag)
}

func (envelope *Envelope) ValidateWith(p *Parameters) error {
	if p == nil {
		return nil
	}
	if p.Tag != nil && envelope.Tag != *p.Tag {
		return asn1core.NewUnexpectedError(*p.Tag, envelope.Tag, "tag mismatch")
	}
	if p.Class != nil && envelope.Class != *p.Class {
		return asn1core.NewUnexpectedError(*p.Class, envelope.Class, "class mismatch")
	}
	return nil
}

func (envelope *Envelope) UpdatePer(p *Parameters) error {
	if p == nil {
		return nil
	}
	if p.Tag != nil {
		envelope.Tag = *p.Tag
	}
	if p.Class != nil {
		envelope.Class = *p.Class
	}
	return nil
}
