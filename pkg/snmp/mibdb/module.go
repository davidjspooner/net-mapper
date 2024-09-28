package mibdb

import (
	"context"
	"io"
	"os"
	"slices"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type reference struct {
	item, module string
}
type Module struct {
	name        string
	imports     map[string]reference
	exports     []mibtoken.Token
	definitions map[string]Definition
}

func (module *Module) Name() string {
	return module.name
}

func (module *Module) Compile(ctx context.Context) error {
	return asn1core.NewUnimplementedError("mib.Compile").MaybeLater()
}

func (module *Module) Exports() (map[string]Definition, error) {
	exports := make(map[string]Definition)
	if module.exports == nil {
		for name, def := range module.definitions {
			exports[name] = def
			exports[module.name+"."+name] = def
		}
	} else {
		for _, tok := range module.exports {
			name := tok.String()
			def, ok := module.definitions[name]
			if !ok {
				return nil, tok.Errorf("exported name %s not found", name)
			}
			exports[name] = def
			exports[module.name+"."+name] = def
		}
	}
	return exports, nil
}

func (module *Module) readOuter(ctx context.Context, s mibtoken.Queue) error {
	if module.definitions == nil {
		module.definitions = make(map[string]Definition)
	}
	ident, err := s.Pop()
	module.name = ident.String()
	if err != nil {
		return err
	}
	err = s.PopExpected("DEFINITIONS", "::=", "BEGIN")
	if err != nil {
		return err
	}

	err = module.readInner(ctx, s)
	if err != nil {
		return err
	}
	return nil
}

func (module *Module) readInner(ctx context.Context, s mibtoken.Queue) error {
	for {
		name, err := s.Pop()
		if err != nil {
			return err
		}
		switch name.String() {
		case "END", "": //be generous with the end of file
			return nil
		case "IMPORTS":
			err = module.readImports(s)
			if err != nil {
				return err
			}
		case "EXPORTS":
			err = module.readExports(s)
			if err != nil {
				return err
			}
		default:
			metaTokens, err := s.PopUntil("::=")
			if err != nil {
				return err
			}
			if metaTokens.Length() > 0 {
				Type, _ := metaTokens.LookAhead(0)
				if Type.String() == "MACRO" {
					mibMacro := &MacroDefintion{metaTokens: metaTokens, source: *name.Source()}
					err = mibMacro.read(ctx, s)
					if err != nil {
						return err
					}
					module.definitions[name.String()] = mibMacro
					continue
				}
			}
			peek, err := s.LookAhead(0)
			if err != nil {
				return nil
			}
			peekStr := peek.String()

			if peekStr == "{" {
				oid := &Oid{metaTokens: metaTokens, source: *name.Source()}
				err = oid.read(ctx, s)
				if err != nil {
					return err
				}
				module.definitions[name.String()] = oid
				continue
			}

			ttype := peek.Type()
			if ttype == mibtoken.STRING || ttype == mibtoken.NUMBER {
				mibType := &Value{metaTokens: metaTokens, source: *name.Source()}
				err = mibType.read(ctx, s)
				if err != nil {
					return err
				}
				module.definitions[name.String()] = mibType
				continue
			}

			if peekStr == "[" || slices.Contains(simpleTypeNames, peekStr) {
				mibType := &SimpleType{metaTokens: metaTokens, source: *name.Source()}
				err = mibType.read(ctx, s)
				if err != nil {
					return err
				}
				module.definitions[name.String()] = mibType
				continue
			}

			mibMacroInvocation := &MacroInvocation{metaTokens: metaTokens, source: *name.Source()}
			mibMacroInvocation.use, err = Lookup[*MacroDefintion](ctx, peekStr)
			if err != nil {
				return name.WrapError(err)
			}

			s.Pop()
			err = mibMacroInvocation.read(ctx, s)
			if err != nil {
				return err
			}
			module.definitions[name.String()] = mibMacroInvocation
		}
	}
}

func (module *Module) readImports(s mibtoken.Queue) error {
	module.imports = make(map[string]reference)
	tokens, err := s.PopUntil(";")
	if err != nil {
		return err
	}
	for !tokens.IsEOF() {
		token, _ := tokens.Pop()
		items := []string{token.String()}
	innerLoop:
		for !tokens.IsEOF() {
			token, _ = tokens.Pop()
			if token.String() == "," {
				token, err = tokens.Pop()
				if err != nil {
					return err
				}
				items = append(items, token.String())
			} else if token.String() == "FROM" {
				token, err = tokens.Pop()
				if err != nil {
					return err
				}
				from := token.String()
				for _, item := range items {
					module.imports[item] = reference{item: item, module: from}
					module.imports[from+"."+item] = reference{item: item, module: from}
				}
				break innerLoop
			} else {
				return token.Errorf("unexpected token %s", token.String())
			}
		}
	}

	return nil
}

func (module *Module) readExports(s mibtoken.Queue) error {
	tokens, err := s.PopUntil(";")
	if err != nil {
		return err
	}
	for !tokens.IsEOF() {
		token, err := tokens.Pop()
		if err != nil {
			return err
		}
		if token.String() != "," {
			switch token.Type() {
			case mibtoken.IDENT, mibtoken.KEYWORD:
				//pass
			default:
				return token.Errorf("unexpected token %s", token.String())
			}
			module.exports = append(module.exports, *token)
		}
	}
	return nil
}

func newScanner(r io.Reader, sourceName string) (*mibtoken.Scanner, error) {
	s, err := mibtoken.NewScanner(r, mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT), mibtoken.WithSource(sourceName))
	if err != nil {
		return nil, err
	}
	return s, nil
}

func ReadModuleFromFile(ctx context.Context, filename string) (*Module, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s, err := newScanner(f, filename)
	if err != nil {
		return nil, err
	}
	module := &Module{}
	ctx = withContext(ctx, func(ctx context.Context, name string) (Definition, error) {
		def, ok := module.definitions[name]
		if !ok {
			return nil, asn1core.NewUnimplementedError("definition %s not found", name)
		}
		return def, nil
	})
	for !s.IsEOF() && ctx.Err() == nil {
		err := module.readOuter(ctx, s)
		if err != nil {
			return nil, err
		}
	}
	return module, s.Err()
}
