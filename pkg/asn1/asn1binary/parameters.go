package asn1binary

import (
	"strconv"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
)

type Parameters struct {
	Tag   *asn1core.Tag
	Class *asn1core.Class
}

func PtrToTag(v asn1core.Tag) *asn1core.Tag {
	return &v
}

func PtrToClass(v asn1core.Class) *asn1core.Class {
	return &v
}

func (p *Parameters) String() string {
	var parts []string
	if p.Tag != nil {
		parts = append(parts, p.Tag.String())
	}
	if p.Class != nil {
		parts = append(parts, p.Class.String())
	}
	return strings.Join(parts, ",")
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
				tag := asn1core.Tag(n)
				params.Tag = &tag
				continue
			case "class":
				n, err := strconv.ParseInt(part[equals+1:], 10, 16)

				if err != nil {
					return nil, err
				}
				class := asn1core.Class(n)
				params.Class = &class
				continue
			}
		} else {
			switch strings.ToLower(part) {
			case "constructed":
				constructed = true
				continue
			}
			tag, err := asn1core.ParseTag(part)
			if err == nil {
				params.Tag = &tag
				continue
			}
			class, err := asn1core.ParseClass(part)
			if err == nil {
				params.Class = &class
				continue
			}
		}

		return nil, asn1core.NewErrorf("unknown parameter %q", part)
	}
	if constructed {
		if params.Tag == nil {
			return nil, asn1core.NewErrorf("constructed parameter requires a tag")
		}
		*params.Tag |= 0x20
	}
	return params, nil
}
