package mibdb

import (
	"context"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Database struct {
	modules   map[string]*Module
	filenames []string
	root      oidBranch
}

const builtInModuleName = "<builtin>"

func New() *Database {
	d := &Database{
		modules: make(map[string]*Module),
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

	for _, simpleType := range simpleTypeNames {
		builtin.definitions[simpleType] = &SimpleType{ident: mibtoken.New(simpleType, builtInPosition)}
	}
	builtin.definitions["iso"] = d.MustReadBuiltInValue(ctx, mibtoken.Object_Identifier, "{ 1 }")
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
	var errList asn1core.ErrorList

	var compile []*Module
	var failedCompile []*Module

	for _, module := range d.modules {
		compile = append(compile, module)
	}

	progress := true
	for progress {
		failedCompile = nil
		errList = nil
		if len(compile) == 0 {
			break
		}
		progress = false
		for _, module := range compile {
			err := module.compile(ctx)
			if err != nil {
				failedCompile = append(failedCompile, module)
				errList = append(errList, err)
			} else {
				progress = true
			}
		}
		compile = failedCompile
	}
	if len(errList) > 0 {
		return errList
	}
	return nil
}

func (d *Database) readDefintions(ctx context.Context) error {
	var errList asn1core.ErrorList
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

	//read all the mibs ( but dont try and compile them yet)
	err := d.readDefintions(ctx)
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
	return nil
}

func (d *Database) FindOID(oid asn1go.OID) (string, Definition, asn1go.OID) {
	return d.root.findOID(oid)
}

func readValue(ctx context.Context, typeName string, s mibtoken.Reader) (Value, error) {
	valueType, _, err := Lookup[Type](ctx, typeName)
	if err != nil {
		return nil, err
	}
	value, err := valueType.readValue(ctx, s)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (d *Database) MustReadBuiltInValue(ctx context.Context, valueTypeName, text string) Value {
	r := strings.NewReader(text)
	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))

	if err != nil {
		panic(err)
	}
	value, err := readValue(ctx, valueTypeName, s)
	if err != nil {
		panic(err)
	}
	return value
}
