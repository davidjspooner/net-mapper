package asn1binary

import (
	"fmt"
	"strings"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
	"golang.org/x/exp/constraints"
)

type mappingPair[T constraints.Integer] struct {
	name string
	val  T
}

type mapping[T constraints.Integer] struct {
	valMap  map[T]*mappingPair[T]
	nameMap map[string]*mappingPair[T]
}

func (m *mapping[T]) AddAlias(name string, aliases ...string) {
	known, ok := m.nameMap[strings.ToLower(name)]
	if !ok {
		panic("unknown name in alias")
	}
	for _, alias := range aliases {
		aliasL := strings.ToLower(alias)
		if _, ok := m.nameMap[aliasL]; ok {
			panic(fmt.Sprintf("duplicate alias %q", alias))
		}
		m.nameMap[aliasL] = known
	}
}

func (m *mapping[T]) Add(name string, val T) {
	nameL := strings.ToLower(name)
	if m.valMap == nil {
		m.valMap = make(map[T]*mappingPair[T])
	}
	if m.nameMap == nil {
		m.nameMap = make(map[string]*mappingPair[T])
	}
	if _, ok := m.valMap[val]; ok {
		panic("duplicate value")
	}
	if _, ok := m.nameMap[nameL]; ok {
		panic("duplicate name")
	}
	p := &mappingPair[T]{name, val}
	m.valMap[val] = p
	m.nameMap[nameL] = p
}

func (m *mapping[T]) Name(val T) (string, error) {
	p, ok := m.valMap[val]
	if !ok {
		return "", asn1error.NewErrorf("unknown value %d", val)
	}
	return p.name, nil
}

func (m *mapping[T]) Value(name string) (T, error) {
	lname := strings.ToLower(name)
	p, ok := m.nameMap[lname]
	if !ok {
		var null T
		return null, asn1error.NewErrorf("unknown name %s", name)
	}
	return p.val, nil
}
