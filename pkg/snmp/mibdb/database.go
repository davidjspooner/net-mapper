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
                                        -- must conform to
                                        -- RFC1155's ObjectSyntax
                                "SYNTAX" type(ObjectSyntax)
                                "MAX-ACCESS" Access
                                "STATUS" Status
                                DescrPart
                                ReferPart
                                IndexPart
                                DefValPart
              VALUE NOTATION ::= value (VALUE ObjectName)
        
              Access ::= "read-only"
                              | "read-write"
                              | "write-only"
                              | "not-accessible"
              Status ::= "mandatory"
                              | "optional"
                              | "obsolete"
                              | "deprecated"
							  | "current"
        
              DescrPart ::=
                         "DESCRIPTION" value (description DisplayString)
                              | empty
        
              ReferPart ::=
                         "REFERENCE" value (reference DisplayString)
                              | empty
        
              IndexPart ::=
                         "INDEX" "{" IndexTypes "}"
                              | empty
              IndexTypes ::=
                         IndexType | IndexTypes "," IndexType
   			  IndexType ::=
							"IMPLIED" Index
							| Index

			  Index ::=
						-- use the SYNTAX value of the
						-- correspondent OBJECT-TYPE invocation
						value(ObjectName)        
              DefValPart ::=
                         "DEFVAL" "{" value (defvalue ObjectSyntax) "}"
                              | empty
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
                         value (vartype ObjectName)
        
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
		for name, def := range module.definitions {
			oid, ok := def.(*OidValue)
			if !ok {
				continue
			}
			d.root.addDefinition(oid.compiled, name, oid)
			//TODO handle tables
		}
	}
	d.logger.DebugContext(ctx, "Finished creating index")
	return nil
}

func (d *Database) FindOID(oid asn1go.OID) (*OidBranch, asn1go.OID) {
	return d.root.findOID(oid)
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
