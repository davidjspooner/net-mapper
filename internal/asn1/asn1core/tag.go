package asn1core

import "fmt"

type Tag int

const Constructed = Tag(0x20)

const (
	TagBoolean          = Tag(0x01)
	TagInteger          = Tag(0x02)
	TagBitString        = Tag(0x03)
	TagOctetString      = Tag(0x04)
	TagNull             = Tag(0x05)
	TagOID              = Tag(0x06)
	TagObjectDescriptor = Tag(0x07)
	TagEnum             = Tag(0x0A)
	TagUTF8String       = Tag(0x0C)
	TagTime             = Tag(0x0E)
	TagSequence         = Tag(0x10)
	TagSet              = Tag(0x11)
	TagNumericString    = Tag(0x12)
	TagPrintableString  = Tag(0x13)
	TagT61String        = Tag(0x14)
	TagIA5String        = Tag(0x16)
	TagUTCTime          = Tag(0x17)
	TagGeneralizedTime  = Tag(0x18)
	TagGeneralString    = Tag(0x1B)
	TagBMPString        = Tag(0x1E)
	TagDate             = Tag(0x1F)

// TagSequenceOf       = TagSequence | Constructed
)

var tagMap mapping[Tag]

func init() {
	tagMap.Add("Boolean", TagBoolean)
	tagMap.Add("Integer", TagInteger)
	tagMap.Add("BitString", TagBitString)
	tagMap.Add("OctetString", TagOctetString)
	tagMap.Add("Null", TagNull)
	tagMap.Add("OID", TagOID)
	tagMap.Add("Enum", TagEnum)
	tagMap.Add("UTF8String", TagUTF8String)
	tagMap.Add("Sequence", TagSequence)
	tagMap.Add("Set", TagSet)
	tagMap.Add("NumericString", TagNumericString)
	tagMap.Add("PrintableString", TagPrintableString)
	tagMap.Add("T61String", TagT61String)
	tagMap.Add("IA5String", TagIA5String)
	tagMap.Add("UTCTime", TagUTCTime)
	tagMap.Add("GeneralizedTime", TagGeneralizedTime)
	tagMap.Add("GeneralString", TagGeneralString)
	tagMap.Add("BMPString", TagBMPString)
	tagMap.Add("Date", TagDate)
	tagMap.Add("Time", TagTime)
	tagMap.Add("ObjectDescriptor", TagObjectDescriptor)

	tagMap.AddAlias("Sequence", "SequenceOf")
	tagMap.AddAlias("Set", "SetOf")
	//tagMap.Add("SequenceOf", TagSequenceOf)
}

func (t Tag) String() string {
	name, err := tagMap.Name(t)
	if err == nil {
		return name
	}
	constructed, baseTag := t.IsConstructed()
	if constructed {
		name, err := tagMap.Name(baseTag)
		if err == nil {
			return name + fmt.Sprintf("( constructed 0x%02X )", int(t))
		}
	}
	return fmt.Sprintf("tag=%02X", int(t))
}

func (t Tag) IsConstructed() (bool, Tag) {
	return t&Constructed != 0, t &^ Constructed
}


func ParseTag(tag string) (Tag, error) {
	return tagMap.Value(tag)
}