package mibdb

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

type DefinitionIndex map[string]Definition

type OidBranch struct {
	parent   *OidBranch
	name     string
	def      Definition
	children map[int]*OidBranch
}

func (branch *OidBranch) Name() string {
	return branch.name
}

func (branch *OidBranch) Definition() Definition {
	return branch.def
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

func (branch *OidBranch) addDefinition(oid asn1go.OID, name string, def Definition) {
	if len(oid) == 0 {
		branch.def = def
		branch.name = name
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
	child.addDefinition(oid[1:], name, def)
}
