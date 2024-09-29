package mibdb

import (
	"context"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1core"
	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

type Database struct {
	modules     map[string]*Module
	filenames   []string
	definitions map[string]Definition
}

func New() *Database {
	d := &Database{
		definitions: make(map[string]Definition),
		modules:     make(map[string]*Module),
	}
	for _, simpleType := range simpleTypeNames {
		d.definitions[simpleType] = &SimpleType{ident: mibtoken.New(simpleType, builtInPosition)}
	}
	d.definitions["iso"] = d.MustReadBuiltInValue(mibtoken.Object_Identifier, "{ 1 }")
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

func (d *Database) withContext(ctx context.Context) context.Context {
	return withContext(ctx, func(ctx context.Context, name string) (Definition, error) {
		def, ok := d.definitions[name]
		if !ok {
			return nil, fmt.Errorf("unknown definition %s", name)
		}
		return def, nil
	})
}

func (d *Database) compileValues(ctx context.Context) error {
	//compile all the values now that we have read them all
	var errList asn1core.ErrorList

	var compile []Value
	var failedCompile []Value

	for _, def := range d.definitions {
		value, ok := def.(Value)
		if ok {
			compile = append(compile, value)
		}
	}

	progress := true
	for progress {
		failedCompile = nil
		errList = nil
		if len(compile) == 0 {
			break
		}
		progress = false
		for _, value := range compile {
			err := value.compile(ctx)
			if err != nil {
				failedCompile = append(failedCompile, value)
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
	for progress {
		errList = nil
		progress = false
		for _, f := range d.filenames {
			if _, ok := d.modules[f]; ok {
				continue
			}
			module, err := ReadModuleFromFile(ctx, f)
			if err != nil {
				errList = append(errList, err)
				continue
			}
			d.modules[module.Name()] = module
			exports, err := module.Exports()
			if err != nil {
				errList = append(errList, err)
				continue
			}
			for name, def := range exports {
				d.definitions[name] = def
			}
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

	ctx = d.withContext(ctx)

	//read all the mibs ( but dont try and compile them yet)
	d.modules = make(map[string]*Module)

	err := d.readDefintions(ctx)
	if err != nil {
		return err
	}
	err = d.compileValues(ctx)
	if err != nil {
		return err
	}
	return nil
}

//func (d *Database) MustReadBuiltInType(text string) *SimpleType {
//	mibType := &SimpleType{}
//	r := strings.NewReader(text)
//	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))
//
//	if err != nil {
//		panic(err)
//	}
//	ctx := d.withContext(context.Background())
//	err = mibType.read(ctx, s)
//	if err != nil {
//		panic(err)
//	}
//	return mibType
//}

func (d *Database) MustReadBuiltInValue(valueTypeName, text string) Value {
	r := strings.NewReader(text)
	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))

	if err != nil {
		panic(err)
	}
	ctx := d.withContext(context.Background())

	valueType, err := Lookup[Type](ctx, valueTypeName)
	if err != nil {
		panic(err)
	}
	err = valueType.compile(ctx)
	if err != nil {
		panic(err)
	}
	value, err := valueType.readValue(ctx, s)
	if err != nil {
		panic(err)
	}
	return value
}
