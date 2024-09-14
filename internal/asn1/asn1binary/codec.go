package asn1binary

import (
	"io"
)

type Encoder interface {
	Encode(i any) error
	EncodeWithParams(i any, params *Parameters) error
}

type Decoder interface {
	Decode(i any) error
	DecodeWithParams(i any, params *Parameters) error
}

type encoder struct {
	w io.Writer
}

type decoder struct {
	r io.Reader
}

func NewEncoder(w io.Writer) Encoder {
	return &encoder{w: w}
}

func NewDecoder(r io.Reader) Decoder {
	return &decoder{r: r}
}

func (e *encoder) EncodeWithParams(i any, params *Parameters) error {
	value := &Value{}
	err := value.PackFromGoWithParameters(i, params)
	if err != nil {
		return err
	}

	bytes, err := value.Marshal()
	if err != nil {
		return err
	}
	_, err = e.w.Write(bytes)
	return err
}

func (e *encoder) Encode(i any) error {
	return e.EncodeWithParams(i, nil)
}

func (d *decoder) DecodeWithParams(i any, params *Parameters) error {
	raw := &Value{}
	_, err := raw.ReadFrom(d.r)
	if err != nil {
		return err
	}
	err = raw.UnpackIntoGoWithParameters(i, params)
	if err != nil {
		return err
	}
	return err
}

func (d *decoder) Decode(i any) error {
	return d.DecodeWithParams(i, nil)
}
