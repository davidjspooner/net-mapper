package mibdb

import (
	"context"
	"fmt"
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

var builtInPosition = mibtoken.Source{Filename: "<BUILTIN>"}

type Definition interface {
	Source() mibtoken.Source
}

type TypeDefinition interface {
	Read(ctx context.Context, name string, meta *mibtoken.List, s *mibtoken.Scanner) (Definition, error)
}

type TypeDefinitionFunc func(ctx context.Context, name string, meta *mibtoken.List, s *mibtoken.Scanner) (Definition, error)

func (f TypeDefinitionFunc) Read(ctx context.Context, name string, meta *mibtoken.List, s *mibtoken.Scanner) (Definition, error) {
	return f(ctx, name, meta, s)
}

type MibDefinedType interface {
	Definition
	TypeDefinition
	Initialize(ctx context.Context, name string, meta *mibtoken.List, s *mibtoken.Scanner) error
}

type Definer[T MibDefinedType] struct {
}

func (mdd *Definer[T]) Read(ctx context.Context, name string, meta *mibtoken.List, s *mibtoken.Scanner) (Definition, error) {
	var t T
	t = reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
	err := t.Initialize(ctx, name, meta, s)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (mdd *Definer[T]) Source() mibtoken.Source {
	return builtInPosition
}

type lookupFunc func(ctx context.Context, name string) (Definition, *Module, error)

type lookuper struct {
	prev *lookuper
	fn   lookupFunc
}

var lookupKey = &struct{}{}

func withLookupContext(ctx context.Context, fn lookupFunc) context.Context {
	l := &lookuper{
		fn: fn,
	}
	if v := ctx.Value(lookupKey); v != nil {
		l.prev = v.(*lookuper)
	}
	ctx = context.WithValue(ctx, lookupKey, l)
	return ctx
}

func Lookup[T any](ctx context.Context, name string) (T, *Module, error) {
	var null T
	if v := ctx.Value(lookupKey); v != nil {
		l := v.(*lookuper)
		for l != nil {
			d, m, err := l.fn(ctx, name)
			if err == nil && d != nil {
				t, ok := d.(T)
				if ok {
					return t, m, nil
				}
				return null, m, fmt.Errorf("definition %T does not implment %s", d, reflect.TypeFor[T]().String())
			}
			l = l.prev
		}
	}
	return null, nil, fmt.Errorf("unknown definition %q", name)
}

type depth struct {
	count map[Definition]int
}

func (d *depth) Inc(def Definition) int {
	if d.count == nil {
		d.count = make(map[Definition]int)
	}
	d.count[def]++
	return d.count[def]
}

func (d *depth) Dec(def Definition) int {
	if d.count == nil {
		return 0
	}
	d.count[def]--
	return d.count[def]
}

var key = &depth{}

func withDepthContect(ctx context.Context) context.Context {
	return context.WithValue(ctx, key, &depth{})
}

func getDepth(ctx context.Context) *depth {
	if v := ctx.Value(key); v != nil {
		return v.(*depth)
	}
	return nil
}
