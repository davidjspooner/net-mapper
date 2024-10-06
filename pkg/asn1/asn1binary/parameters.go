package asn1binary

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

type Parameters struct {
	Tag   *Tag
	Class *Class
}

func PtrToTag(v Tag) *Tag {
	return &v
}

func PtrToClass(v Class) *Class {
	return &v
}

func (p *Parameters) String() string {
	var parts []string
	universal := false
	if p.Class != nil {
		parts = append(parts, p.Class.String())
		universal = *p.Class == ClassUniversal
	}
	if p.Tag != nil {
		s, err := tagMap.Name(*p.Tag)
		if err == nil && universal {
			parts = append(parts, s)
		} else {
			parts = append(parts, fmt.Sprintf("tag=%d", *p.Tag))
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func ParseParameters(s string) (*Parameters, error) {
	params := &Parameters{}
	parts := strings.Split(s, ",")
	constructed := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		equals := strings.Index(part, "=")
		if equals != -1 {
			key := strings.ToLower(part[:equals])

			switch key {
			case "tag":
				n, err := strconv.ParseInt(part[equals+1:], 10, 16)
				if err != nil {
					return nil, err
				}
				tag := Tag(n)
				params.Tag = &tag
				continue
			case "class":
				n, err := strconv.ParseInt(part[equals+1:], 10, 16)

				if err != nil {
					return nil, err
				}
				class := Class(n)
				params.Class = &class
				continue
			}
		} else {
			switch strings.ToLower(part) {
			case "constructed":
				constructed = true
				continue
			}
			tag, err := ParseTag(part)
			if err == nil {
				params.Tag = &tag
				continue
			}
			class, err := ParseClass(part)
			if err == nil {
				params.Class = &class
				continue
			}
		}

		return nil, asn1error.NewErrorf("unknown parameter %q", part)
	}
	if constructed {
		if params.Tag == nil {
			return nil, asn1error.NewErrorf("constructed parameter requires a tag")
		}
		*params.Tag |= 0x20
	}
	return params, nil
}

func (p *Parameters) Validate(envelope *Envelope) error {
	if p == nil {
		return nil
	}
	if p.Tag != nil && envelope.Tag != *p.Tag {
		return asn1error.NewUnexpectedError(*p.Tag, envelope.Tag, "tag mismatch")
	}
	if p.Class != nil && envelope.Class != *p.Class {
		return asn1error.NewUnexpectedError(*p.Class, envelope.Class, "class mismatch")
	}
	return nil
}

func (p *Parameters) Update(envelope *Envelope) error {
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
