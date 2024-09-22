package asn1mib

import (
	"fmt"
	"os"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/internal/asn1/asn1go"
)

type mibFile struct {
	filename    string
	mibname     Token
	err         error
	exports     []string
	definitions map[string]Definition
}

type Directory struct {
	files       []*mibFile
	definitions map[string]Definition
}

func NewDirectory() *Directory {
	d := &Directory{
		definitions: make(map[string]Definition),
	}
	d.definitions["OBJECT IDENTIFIER"] = &oidReader{}
	d.definitions["MACRO"] = &macroDefintionReader{}
	d.definitions["iso"] = &oidDefintion{
		oid:    []int{1},
		source: builtInPosition,
	}
	return d
}

func (d *Directory) AddFile(filename string) error {
	d.files = append(d.files, &mibFile{filename: filename, definitions: make(map[string]Definition)})
	return nil
}

func (d *Directory) CreateIndex() error {

	var todo, failed, succeded []*mibFile

	todo = append(todo, d.files...)

	progress := true
	for progress {
		failed = nil
		if len(todo) == 0 {
			break
		}
		progress = false
		for _, f := range todo {
			f.err = d.tryReadMib(f)
			if f.err != nil {
				failed = append(failed, f)
			} else {
				succeded = append(succeded, f)
				progress = true
			}
		}
		todo = failed
	}
	_ = succeded //for debugging
	if len(failed) > 0 {
		err := asn1core.ErrorList{}
		for _, mf := range failed {
			err = append(err, mf.err)
		}
		return err
	}
	return nil
}

func (d *Directory) tryReadMib(mf *mibFile) error {

	mf.exports = nil

	f, err := os.Open(mf.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := NewScanner(f, WithSkip(WHITESPACE, COMMENT), WithFilename(mf.filename))
	if err != nil {
		return err
	}
	for s.Scan() {
		err = d.readDefintion(mf, s)
		if err != nil {
			return err
		}
	}

	if mf.exports == nil {
		for name, def := range mf.definitions {
			d.definitions[name] = def
		}
	} else {
		for _, name := range mf.exports {
			def, ok := mf.definitions[name]
			if !ok {
				return s.position().Errorf("unknown export %s", name)
			}
			d.definitions[name] = def
		}
	}

	return s.Err()
}

func (d *Directory) readDefintion(mf *mibFile, s *Scanner) error {
	ident := s.LookAhead(0)
	position := ident.Source()
	s.Scan()
	err := s.PopExpected("DEFINITIONS", "::=", "BEGIN")
	if err != nil {
		return err
	}
	mf.mibname = ident
	for {
		tok := s.LookAhead(0)
		switch tok.String() {
		case "END", "": //be generous with the end of file
			return nil
		case "IMPORTS":
			err = d.readImports(s)
			if err != nil {
				return err
			}
		case "EXPORTS":
			exports, err := s.PopUntil(";") //discard
			if err != nil {
				return err
			}
			for _, e := range exports {
				t := e.String()
				if t != "," && t != ";" {
					mf.exports = append(mf.exports, e.String())
				}
			}
		default:
			defintionPosition := tok.source
			name := tok.String()
			if !s.Scan() {
				return position.Errorf("unterminated DEFINITION::=BEGIN")
			}
			var newDefintion Definition
			if s.LookAhead(0).String() == "::=" {
				reader := &simpleTypeDefintionReader{}
				newDefintion, err = reader.ReadDefinition(name, d, s)
				if err != nil {
					return err
				}
				d.definitions[name] = newDefintion
				mf.definitions[name] = newDefintion
				continue
			}
			defType := s.Pop()
			if defType.Type() != IDENT {
				return defType.Errorf("unexpected %q in DEFINITION::=BEGIN", s.LookAhead(0))
			}
			defintion, ok := d.definitions[defType.String()]
			if !ok {
				return defintionPosition.Errorf("unknown definition type %s", defType.String())
			}
			reader, ok := defintion.(TypeDefinition)
			if !ok {
				return defintionPosition.Errorf("definition type %s is not a reader", defType.String())
			}
			newDefintion, err := reader.Read(name, d, s)
			if err != nil {
				return err
			}
			d.definitions[name] = newDefintion
			mf.definitions[name] = newDefintion
		}
	}
}

func (d *Directory) readImports(s *Scanner) error {
	//startPosition := *s.position()
	err := s.PopExpected("IMPORTS")
	if err != nil {
		return err
	}
	for {
		var dependencies TokenList
		firsttok := s.Pop()
		if firsttok.String() == ";" {
			return nil
		}
		dependancyPosition := firsttok.source
		dependencies = append(dependencies, firsttok)
	innerloop:
		for {
			tok := s.Pop()
			switch tok.String() {
			case "FROM":
				otherModule, err := s.PopIdent()
				if err != nil {
					return err
				}
				for _, dep := range dependencies {
					_, ok := d.definitions[dep.String()]
					if !ok {
						return dependancyPosition.Errorf("needs %q from %q", dep.String(), otherModule.String())
					}
				}
				dependencies = nil
				break innerloop
			case ",":
				dependancy, err := s.PopIdent()
				if err != nil {
					return err
				}
				dependencies = append(dependencies, dependancy)
			default:
				return dependancyPosition.Errorf("unexpected %q in IMPORTS", tok.String())
			}
		}
	}
}

func (d *Directory) OIDLookup(s string) (asn1go.OID, error) {
	definition, ok := d.definitions[s]
	if !ok {
		return nil, fmt.Errorf("unknown OID %s", s)
	}
	oidDefintion, ok := definition.(OIDValue)
	if !ok {
		return nil, fmt.Errorf("definition %s (%T) is not an OID", s, definition)
	}
	return oidDefintion.OID(), nil
}
