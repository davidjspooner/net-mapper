package mibdb

import (
	"context"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Database struct {
	modules   map[string]*Module
	filenames []string
	root      OidBranch
	logger    *slog.Logger
}

const builtInModuleName = "<builtin>"

func New(logger *slog.Logger) *Database {

	logger = logger.WithGroup("mibdb")

	d := &Database{
		modules: make(map[string]*Module),
		logger:  logger,
	}

	builtin := &Module{
		database:    d,
		name:        builtInModuleName,
		imports:     make(map[string]reference),
		exports:     nil,
		definitions: make(map[string]Definition),
	}
	d.modules["<builtin>"] = builtin

	ctx := builtin.withContext(context.Background())
	ctx = withDepthContect(ctx)

	for _, simpleType := range simpleTypeNames {
		builtin.definitions[simpleType] = &TypeReference{ident: mibtoken.New(simpleType, builtInPosition)}
	}
	builtin.definitions["iso"] = d.MustReadBuiltInValue(ctx, mibtoken.Object_Identifier, "{ 1 }")
	builtin.definitions["OBJECT-TYPE"] = d.MustReadMacroDefinition(ctx, "OBJECT-TYPE", `
		BEGIN
			TYPE NOTATION ::=
						"SYNTAX" Syntax
						UnitsPart
						"MAX-ACCESS" Access
						"STATUS" Status
						"DESCRIPTION" Text
						ReferPart
						IndexPart
						DefValPart

			VALUE NOTATION ::=
						value(VALUE ObjectName)

			Syntax ::=   -- Must be one of the following:
							-- a base type (or its refinement),
							-- a textual convention (or its refinement), or
							-- a BITS pseudo-type
						type
						| "BITS" "{" NamedBits "}"

			NamedBits ::= NamedBit
						| NamedBits "," NamedBit

			NamedBit ::=  identifier "(" number ")" -- number is nonnegative

			UnitsPart ::=
						"UNITS" Text
						| empty

			Access ::=
						"not-accessible"
						| "accessible-for-notify"
						| "read-only"
						| "read-write"
						| "read-create"

			Status ::=
						"current"
						| "deprecated"
						| "obsolete"
						| "mandatory"

			ReferPart ::=
						"REFERENCE" Text
						| empty

			IndexPart ::=
						"INDEX"    "{" IndexTypes "}"
						| "AUGMENTS" "{" Entry      "}"
						| empty
			IndexTypes ::=
						IndexType
						| IndexTypes "," IndexType
			IndexType ::=
						"IMPLIED" Index
						| Index

			Index ::=
							-- use the SYNTAX value of the
							-- correspondent OBJECT-TYPE invocation
						value(OBJECT IDENTIFIER)
			Entry ::=
							-- use the INDEX value of the
							-- correspondent OBJECT-TYPE invocation
						value(OBJECT IDENTIFIER)

			DefValPart ::= "DEFVAL" "{" Defvalue "}"
						| empty

			Defvalue ::=  -- must be valid for the type specified in
						-- SYNTAX clause of same OBJECT-TYPE macro
						value(ObjectSyntax)
						| "{" BitsValue "}"

			BitsValue ::= BitNames
						| empty

			BitNames ::=  BitName
						| BitNames "," BitName

			BitName ::= identifier

			-- a character string as defined in section 3.1.1
			Text ::= value(IA5String)
		END
	`)

	builtin.definitions["TRAP-TYPE"] = d.MustReadMacroDefinition(ctx, "TRAP-TYPE", `
          BEGIN
              TYPE NOTATION ::= "ENTERPRISE" value
                                    (enterprise OBJECT IDENTIFIER)
                                VarPart
                                DescrPart
                                ReferPart
              VALUE NOTATION ::= value (VALUE INTEGER)
        
              VarPart ::=
                         "VARIABLES" "{" VarTypes "}"
                              | empty
              VarTypes ::=
                         VarType | VarTypes "," VarType
              VarType ::=
                         value (vartype OBJECT IDENTIFIER)
        
              DescrPart ::=
                         "DESCRIPTION" value (description DisplayString)
                              | empty
        
              ReferPart ::=
                         "REFERENCE" value (reference DisplayString)
                              | empty
        
          END
	`)

	d.logger.Debug("Built-in MIB loaded")

	return d
}

func (d *Database) AddFile(filenames ...string) error {
	for _, filename := range filenames {
		if !slices.Contains(d.filenames, filename) {
			d.filenames = append(d.filenames, filename)
		}
	}
	return nil
}

func (d *Database) Logger() *slog.Logger {
	return d.logger
}

func (d *Database) AddDirectory(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		nameL := strings.ToLower(file.Name())
		if strings.HasSuffix(nameL, ".mib") {
			err := d.AddFile(path.Join(dir + file.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Database) compileValues(ctx context.Context) error {
	//compile all the values now that we have read them all
	var errList asn1error.List

	var compile []*Module
	var failedCompile []*Module

	for _, module := range d.modules {
		compile = append(compile, module)
	}

	successCount := 1
	passCount := 0
	for successCount > 0 {
		failedCompile = nil
		errList = nil
		if len(compile) == 0 {
			break
		}
		successCount = 0
		passCount++
		for _, module := range compile {
			//started := time.Now()
			err := module.compileValues(ctx)
			//elapsed := time.Since(started)
			//d.logger.DebugContext(ctx, "Compile module", slog.String("module", module.Name()), slog.Any("elapsed", fmt.Sprintf("%0.3f", elapsed.Seconds())), slog.Any("error", err))
			if err != nil {
				failedCompile = append(failedCompile, module)
				errList = append(errList, err)
			} else {
				successCount++
			}
		}
		compile = failedCompile
		d.logger.DebugContext(ctx, "Compiled values", slog.Any("pass", passCount), slog.Any("success", successCount), slog.Any("deferred", len(failedCompile)))
	}
	if len(errList) > 0 {
		d.logger.DebugContext(ctx, "Failed to compile values", slog.Int("failed", len(errList)))
		return errList
	}
	return nil
}

func (d *Database) readDefintions(ctx context.Context) error {
	var errList asn1error.List
	progress := true
	var done []string
	for progress {
		errList = nil
		progress = false
		for _, f := range d.filenames {
			if slices.Contains(done, f) {
				continue
			}
			module, err := readModuleFromFile(ctx, d, f)
			if err != nil {
				errList = append(errList, err)
				continue
			}
			d.modules[module.Name()] = module
			done = append(done, f)
			progress = true
		}
		if len(errList) == 0 {
			break
		}
	}
	if len(errList) > 0 {
		return errList
	}
	return nil
}

func (d *Database) CreateIndex(ctx context.Context) error {

	ctx = withDepthContect(ctx)

	//read all the mibs ( but dont try and compile them yet)
	err := d.readDefintions(ctx)
	d.logger.DebugContext(ctx, "Finished reading definitions", slog.Any("error", err))
	if err != nil {
		return err
	}

	err = d.compileValues(ctx)
	if err != nil {
		return err
	}

	for _, module := range d.modules {
		for _, def := range module.definitions {
			oid, ok := def.(*Object)
			if !ok {
				continue
			}
			d.root.addDefinition(oid.compiled, oid)
		}
	}

	d.logger.DebugContext(ctx, "Finished creating index")
	return nil
}

func (d *Database) FindOID(oid asn1go.OID) (*OidBranch, asn1go.OID) {
	return d.root.findOID(oid)
}

func (d *Database) LookupName(name string) (Definition, *Module) {
	for _, module := range d.modules {
		def, ok := module.definitions[name]
		if !ok {
			continue
		}
		return def, module
	}
	return nil, nil
}

func (d *Database) MustReadBuiltInValue(ctx context.Context, valueTypeName, text string) Value {
	r := strings.NewReader(text)
	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))
	builtin := d.modules["<builtin>"]
	if err != nil {
		panic(err)
	}
	valueType, _, err := Lookup[Type](ctx, valueTypeName)
	if err != nil {
		panic(err)
	}
	value, err := valueType.readValue(ctx, builtin, s)
	if err != nil {
		panic(err)
	}
	return value
}

func (d *Database) MustReadMacroDefinition(ctx context.Context, name, text string) *MacroDefintion {
	r := strings.NewReader(text)
	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))
	builtin := d.modules["<builtin>"]
	if err != nil {
		panic(err)
	}
	mibMacro := &MacroDefintion{name: name}
	mibMacro.set(builtin, nil, *s.Source())
	err = mibMacro.readDefinition(ctx, builtin, s)
	if err != nil {
		panic(err)
	}
	return mibMacro
}
