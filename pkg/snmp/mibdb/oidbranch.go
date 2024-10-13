package mibdb

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

type DefinitionIndex map[string]Definition

type OidBranch struct {
	Stash
	parent   *OidBranch
	def      *OidValue
	children map[int]*OidBranch
}

func (branch *OidBranch) Value() *OidValue {
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

func (branch *OidBranch) addDefinition(oid asn1go.OID, def *OidValue) {
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
