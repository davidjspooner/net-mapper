package asn1mib

import (
	"context"
	"fmt"

	"github.com/davidjspooner/dsflow/pkg/job"
)

type Directory struct {
	files map[string]*file
}

func NewDirectory() *Directory {
	d := &Directory{
		files: make(map[string]*file),
	}
	return d
}

func (d *Directory) AddFile(filename string) error {
	mf, err := newFile(filename)
	if err != nil {
		return err
	}
	d.files[mf.Name] = mf
	return nil
}

func (d *Directory) importMib(mibName string, tokens []string) error {
	fmt.Printf("Importing MIB %s\n", mibName)
	for _, token := range tokens {
		fmt.Printf("    %s\n", token)
	}
	return nil
}

func (d *Directory) CreateIndex() error {

	graph := job.NewNodeGraph()

	for _, mf := range d.files {
		graph.AddNode(mf)
	}

	var all []string
	for _, mf := range d.files {
		all = append(all, mf.ID())
		for _, imp := range mf.Imports {
			graph.SetPrecursorIDs(mf.ID(), imp.From)
		}
	}

	sequence, err := graph.PlanIDs(all...)
	if err != nil {
		errorList, ok := err.(job.ErrorList)
		if ok {
			for _, e := range errorList {
				fmt.Println(e)
			}
		} else {
			fmt.Println(err)
		}
	}
	ctx := context.Background()
	sequence.Run(ctx, 1, func(ctx context.Context, node *job.NodeWithPrecursors) error {
		err := d.importMib(node.ID(), node.Node().(*file).Tokens)
		return err
	}, job.LoggerFunc(func(format string, v ...any) {
		fmt.Printf(format, v...)
	}))

	return nil
}
