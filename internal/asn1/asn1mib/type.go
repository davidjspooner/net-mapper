package asn1mib

import "reflect"

var builtInPosition = Position{Filename: "<BUILTIN>"}

type Definition interface {
	Source() Position
}

type TypeDefinition interface {
	Read(name string, d *Directory, s *Scanner) (Definition, error)
}

type TypeDefinitionFunc func(name string, d *Directory, s *Scanner) (Definition, error)

func (f TypeDefinitionFunc) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	return f(name, d, s)
}

type MibDefinedType interface {
	Definition
	TypeDefinition
	Initialize(name string, d *Directory, s *Scanner) error
}

type Definer[T MibDefinedType] struct {
}

func (mdd *Definer[T]) Read(name string, d *Directory, s *Scanner) (Definition, error) {
	var t T
	t = reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
	err := t.Initialize(name, d, s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (mdd *Definer[T]) Source() Position {
	return builtInPosition
}
