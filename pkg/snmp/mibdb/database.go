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
	d.definitions["iso"] = d.MustReadBuiltInValue("{ 1 }")
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

func (d *Database) CreateIndex(ctx context.Context) error {

	var compile, failed, succeded []*Module

	ctx = d.withContext(ctx)

	//read all the mibs ( but dont try and compile them yet)
	d.modules = make(map[string]*Module)
	var errList asn1core.ErrorList
	progress := true
	for progress {
		failed = nil
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

	//compile all the mibs now that we have read them all

	for _, module := range d.modules {
		compile = append(compile, module)
	}

	progress = true
	for progress {
		failed = nil
		errList = nil
		if len(compile) == 0 {
			break
		}
		progress = false
		for _, module := range compile {
			err := module.Compile(ctx)
			if err != nil {
				failed = append(failed, module)
				errList = append(errList, err)
			} else {
				succeded = append(succeded, module)
				progress = true
			}
		}
		compile = failed
	}
	_ = succeded //for debugging
	if len(errList) > 0 {
		return errList
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

func (d *Database) MustReadBuiltInValue(text string) *Oid {
	value := &Oid{}
	r := strings.NewReader(text)
	s, err := mibtoken.NewScanner(r, mibtoken.WithSource("<built-in>"), mibtoken.WithSkip(mibtoken.WHITESPACE, mibtoken.COMMENT))

	if err != nil {
		panic(err)
	}
	ctx := d.withContext(context.Background())
	err = value.read(ctx, s)
	if err != nil {
		panic(err)
	}
	return value
}
