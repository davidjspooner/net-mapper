package asn1binary

import (
	"fmt"
)

type Envelope struct {
	Class Class
	Tag   Tag
}

func (e *Envelope) String() string {

	tagStr, err := tagMap.Name(e.Tag)
	if err != nil || e.Class != ClassUniversal {
		tagStr = fmt.Sprintf("tag=%d", e.Tag)
	}
	return fmt.Sprintf("[%s %s]", e.Class, tagStr)
}
