package asn1go

import (
	"strconv"
	"testing"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1binary"
)

type IntegerTest struct {
	Bytes  []byte
	String string
}

func TestInteger(t *testing.T) {
	integerTests := []IntegerTest{
		{Bytes: []byte{0x00}, String: "0"},
		{Bytes: []byte{0x01}, String: "1"},
		{Bytes: []byte{0x7F}, String: "127"},
		{Bytes: []byte{0x80}, String: "-128"},
		{Bytes: []byte{0xFF}, String: "-1"},
		{Bytes: []byte{0x07, 0xE4}, String: "2020"},
	}
	for _, test := range integerTests {
		t.Run("decode to "+test.String, func(t *testing.T) {
			var v Integer
			if err := v.UnpackAsn1(asn1binary.Envelope{Tag: asn1binary.TagInteger}, test.Bytes); err != nil {
				t.Error(err)
			}
			if got := v.String(); got != test.String {
				t.Errorf("got %q, want %q", got, test.String)
			}
		})
		t.Run("encode "+test.String, func(t *testing.T) {
			var v Integer
			n, err := strconv.ParseInt(test.String, 10, 64)
			if err != nil {
				t.Error(err)
			}
			v.SetInt(n)
			//compare v.Bytes with test.Bytes
			if got := string(v); got != string(test.Bytes) {
				t.Errorf("got 0x%X, want 0x%X", got, test.Bytes)
			}
		})
	}
}
