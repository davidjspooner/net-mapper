package asn1mib

import (
	"context"
	"fmt"
	"os"

	"github.com/davidjspooner/net-mapper/internal/asn1/asn1core"
)

type mibFile struct {
	filename    string
	mibname     string
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
	d.definitions[""] = &Definer[*simpleTypeDefintion]{}

	for _, simpleType := range simpleTypeNames {
		d.definitions[simpleType] = &simpleTypeDefintion{typeClass: simpleType}
	}
	d.definitions["MACRO"] = &Definer[*macroDefintion]{}
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

	ctx := context.Background()
	ctx = withContext(ctx, func(ctx context.Context, name string) (Definition, error) {
		d, ok := d.definitions[name]
		if !ok {
			return nil, fmt.Errorf("unknown definition %s", name)
		}
		return d, nil
	})

	progress := true
	for progress {
		failed = nil
		if len(todo) == 0 {
			break
		}
		progress = false
		for _, f := range todo {
			f.err = d.tryReadMib(ctx, f)
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

func (d *Directory) tryReadMib(ctx context.Context, mf *mibFile) error {

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
	for !s.IsEOF() {
		err = d.readDefintion(ctx, mf, s)
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
				return s.Errorf("unknown export %s", name)
			}
			d.definitions[name] = def
		}
	}

	return s.Err()
}

func (d *Directory) readDefintion(ctx context.Context, mf *mibFile, s *Scanner) error {
	ident, err := s.Pop()
	if err != nil {
		return err
	}
	err = s.PopExpected("DEFINITIONS", "::=", "BEGIN")
	if err != nil {
		return err
	}
	mf.mibname = ident.String()
	for {
		tok, err := s.LookAhead(0)
		if err != nil {
			return err
		}
		switch tok.String() {
		case "END", "": //be generous with the end of file
			s.Pop()
			return nil
		case "IMPORTS":
			err = d.readImports(s)
			if err != nil {
				return err
			}
		case "EXPORTS":
			s.Pop()                         //discard EXPORTS;
			exports, err := s.PopUntil(";") //discard
			if err != nil {
				return err
			}
			exports.ForEach(func(tok *Token) error {
				t := tok.String()
				if t != "," && t != ";" {
					mf.exports = append(mf.exports, t)
				}
				return nil
			})
		default:
			name, err := s.PopType(IDENT)
			if err != nil {
				return err
			}
			peek, err := s.LookAhead(0)
			if err != nil {
				return err
			}
			var reader TypeDefinition
			var typeName string
			if peek.IsText("::=") {
				typeName = ""
			} else {
				defType, _ := s.PopType(IDENT)
				typeName = defType.String()
			}
			defintion, ok := d.definitions[typeName]
			if !ok {
				return name.Errorf("unknown definition type %s", typeName)
			}
			reader, ok = defintion.(TypeDefinition)
			if !ok {
				return name.Errorf("definition type %s is not a reader", typeName)
			}
			meta, err := s.PopUntil("::=")
			if err != nil {
				return err
			}
			newDefintion, err := reader.Read(ctx, name.String(), meta, s)
			if err != nil {
				return err
			}
			d.definitions[name.String()] = newDefintion
			mf.definitions[name.String()] = newDefintion
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
		dependencies := &TokenList{
			Filename: s.Source().Filename,
		}
		firsttok, err := s.Pop()
		if err != nil {
			return err
		}
		if firsttok.String() == ";" {
			return nil
		}
		dependancyPosition := firsttok.Source()
		dependencies.AppendTokens(firsttok)
	innerloop:
		for {
			tok, err := s.Pop()
			if err != nil {
				return err
			}
			switch tok.String() {
			case "FROM":
				otherModule, err := s.PopType(IDENT)
				if err != nil {
					return err
				}
				err = dependencies.ForEach(func(dependancy *Token) error {
					_, ok := d.definitions[dependancy.String()]
					if !ok {
						return dependancyPosition.Errorf("needs %q from %q", dependancy.String(), otherModule.String())
					}
					return nil
				})
				if err != nil {
					return err
				}
				break innerloop
			case ",":
				dependancy, err := s.PopType(IDENT)
				if err != nil {
					return err
				}
				dependencies.AppendTokens(dependancy)
			default:
				return dependancyPosition.Errorf("unexpected %q in IMPORTS", tok.String())
			}
		}
	}
}
