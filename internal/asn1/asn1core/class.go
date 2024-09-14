package asn1core

import "fmt"

type Class int

const (
	ClassUniversal       = Class(0)
	ClassApplication     = Class(1)
	ClassContextSpecific = Class(2)
	ClassPrivate         = Class(3)
)

var classMap mapping[Class]

func init() {
	classMap.Add("Universal", ClassUniversal)
	classMap.Add("Application", ClassApplication)
	classMap.Add("ContextSpecific", ClassContextSpecific)
	classMap.Add("Private", ClassPrivate)
}

func (c Class) String() string {
	name, err := classMap.Name(c)
	if err == nil {
		return name
	}
	return fmt.Sprintf("class=%02X", int(c))
}

func ParseClass(class string) (Class, error) {
	return classMap.Value(class)
}
