package mibdb

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type DefinitionIndex map[string]Definition

type OidBranch struct {
	parent   *OidBranch
	def      *Object
	children map[int]*OidBranch
}

func (branch *OidBranch) Object() *Object {
	return branch.def
}

func (branch *OidBranch) ChildValues() []*Object {
	var values []*Object
	keys := maps.Keys(branch.children)
	slices.Sort(keys)
	for _, key := range keys {
		child := branch.children[key]
		if child.def != nil {
			values = append(values, child.def)
		}
	}
	return values
}

func (branch *OidBranch) Parent() *OidBranch {
	return branch.parent
}

func (branch *OidBranch) findOID(oid asn1go.OID) (*OidBranch, asn1go.OID) {
	if len(oid) == 0 {
		return branch, oid
	}
	child, ok := branch.children[oid[0]]
	if !ok {
		return branch, oid
	}
	childBranch, tail := child.findOID(oid[1:])
	if childBranch == nil || childBranch.def == nil {
		return branch, oid
	} else {
		return childBranch, tail
	}
}

func (branch *OidBranch) addDefinition(oid asn1go.OID, def *Object) {
	if len(oid) == 0 {
		branch.def = def
		return
	}
	child, ok := branch.children[oid[0]]
	if !ok {
		child = &OidBranch{
			parent: branch,
		}
		if branch.children == nil {
			branch.children = make(map[int]*OidBranch)
		}
		branch.children[oid[0]] = child
	}
	child.addDefinition(oid[1:], def)
}
