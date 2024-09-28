package asn1binary

type Packer interface {
	PackAsn1(params *Parameters) (Envelope, []byte, error)
}

type Unpacker interface {
	UnpackAsn1(Envelope, []byte) error
}

type PackerFunc func(params *Parameters) (Envelope, []byte, error)

func (f PackerFunc) PackAsn1(params *Parameters) (Envelope, []byte, error) {
	return f(params)
}

type UnpackerFunc func(Envelope, []byte) error

func (f UnpackerFunc) UnpackAsn1(envelope Envelope, bytes []byte) error {
	return f(envelope, bytes)
}

type Transformer interface {
	Packer
	Unpacker
}

type TransformerFuncs struct {
	Pack   PackerFunc
	Unpack UnpackerFunc
}

func (f TransformerFuncs) PackAsn1(params *Parameters) (Envelope, []byte, error) {
	return f.Pack(params)
}
func (f TransformerFuncs) UnpackAsn1(envelope Envelope, bytes []byte) error {
	return f.Unpack(envelope, bytes)
}
