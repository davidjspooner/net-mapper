package asn1binary

import "github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"

type CharSetByteValidator [32 / 4]uint32

func getIndexAndMask(r byte) (int, uint32) {
	return int(r) / 32, 1 << uint32(r%32)
}

func (c *CharSetByteValidator) ValidateBytes(bytes []byte) error {
	for _, r := range bytes {
		n, mask := getIndexAndMask(r)
		if c[n]&mask == 0 {
			return asn1core.NewErrorf("invalid character %q", r)
		}
	}
	return nil
}

func (c *CharSetByteValidator) setChars(set bool, chars ...byte) *CharSetByteValidator {
	for _, r := range chars {
		c.set(set, r)
	}
	return c
}
func (c *CharSetByteValidator) setCharRange(set bool, from, to byte) *CharSetByteValidator {
	for r := from; r <= to; r++ {
		c.set(set, r)
	}
	return c
}

func (c *CharSetByteValidator) set(set bool, r byte) *CharSetByteValidator {
	n, mask := getIndexAndMask(r)
	if set {
		c[n] |= mask
	} else {
		c[n] &^= mask
	}
	return c
}

var PrintableStringValidator, IA5StringValidator CharSetByteValidator

func init() {
	PrintableStringValidator.setCharRange(true, 'A', 'Z').setCharRange(true, 'a', 'z').setCharRange(true, '0', '9').setChars(true, ' ', '\'', '(', ')', '+', ',', '-', '.', '/', ':', '=', '?')
	IA5StringValidator.setCharRange(true, 0, 127)
}
