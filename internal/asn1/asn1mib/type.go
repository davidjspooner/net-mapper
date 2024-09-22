package asn1mib

var builtInPosition = Position{Filename: "<BUILTIN>"}

type Definition interface {
	Source() Position
}

type TypeDefinition interface {
	Read(name string, d *Directory, s *Scanner) (Definition, error)
}
