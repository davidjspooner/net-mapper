package mibdb

import (
	"context"
	"fmt"
	"reflect"

	"github.com/davidjspooner/net-mapper/pkg/snmp/mibtoken"
)

var builtInPosition = mibtoken.Position{Filename: "<BUILTIN>"}

type Definition interface {
	Source() mibtoken.Position
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

func (mdd *Definer[T]) Source() mibtoken.Position {
	return builtInPosition
}

type lookupFunc func(ctx context.Context, name string) (Definition, error)

type lookuper struct {
	prev *lookuper
	fn   lookupFunc
}

var lookupKey = &struct{}{}

func withContext(ctx context.Context, fn lookupFunc) context.Context {
	l := &lookuper{
		fn: fn,
	}
	if v := ctx.Value(lookupKey); v != nil {
		l.prev = v.(*lookuper)
	}
	ctx = context.WithValue(ctx, lookupKey, l)
	return ctx
}

func Lookup[T any](ctx context.Context, name string) (T, error) {
	var null T
	if v := ctx.Value(lookupKey); v != nil {
		l := v.(*lookuper)
		for l != nil {
			d, err := l.fn(ctx, name)
			if err == nil && d != nil {
				t, ok := d.(T)
				if ok {
					return t, nil
				}
				return null, fmt.Errorf("definition %T does not implment %T", d, null)
			}
			l = l.prev
		}
	}
	return null, fmt.Errorf("unknown definition %s", name)
}
