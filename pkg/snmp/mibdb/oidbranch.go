package mibdb

import (
	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1go"
)

type DefinitionIndex map[string]Definition

type oidBranch struct {
	name     string
	def      Definition
	children map[int]*oidBranch
}

func (branch *oidBranch) findOID(oid asn1go.OID) (string, Definition, asn1go.OID) {
	if len(oid) == 0 {
		return branch.name, branch.def, oid
	}
	child, ok := branch.children[oid[0]]
	if !ok {
		return branch.name, branch.def, oid
	}
	name, def, tail := child.findOID(oid[1:])
	if def == nil {
		return branch.name, branch.def, oid
	} else {
		return name, def, tail
	}
}

func (branch *oidBranch) addDefinition(oid asn1go.OID, name string, def Definition) {
	if len(oid) == 0 {
		branch.def = def
		branch.name = name
		return
	}
	child, ok := branch.children[oid[0]]
	if !ok {
		child = &oidBranch{}
		if branch.children == nil {
			branch.children = make(map[int]*oidBranch)
		}
		branch.children[oid[0]] = child
	}
	child.addDefinition(oid[1:], name, def)
}
