package asn1binary

func MarshalWithParams(v interface{}, params *Parameters) ([]byte, error) {
	packer, err := GetPackerFor(v)
	if err != nil {
		return nil, err
	}
	value := Value{}
	value.Envelope, value.Bytes, err = packer.PackAsn1(params)
	if err != nil {
		return nil, err
	}
	bytes, err := value.Marshal()
	return bytes, err
}

func Marshal(v interface{}) ([]byte, error) {
	return MarshalWithParams(v, nil)
}

func Unmarshal(bytes []byte, v interface{}) ([]byte, error) {
	value := Value{}
	tail, err := value.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}
	unpacker, err := GetUnpackerFor(v)
	if err != nil {
		return nil, err
	}
	err = unpacker.UnpackAsn1(value.Envelope, value.Bytes)
	if err != nil {
		return nil, err
	}
	return tail, nil
}
