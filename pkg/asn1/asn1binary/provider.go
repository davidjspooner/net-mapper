package asn1binary

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/davidjspooner/net-mapper/pkg/asn1/asn1error"
)

var lock sync.RWMutex

type PackerProviderFunc func(i any) (Packer, error)
type UnpackerProviderFunc func(i any) (Unpacker, error)

type provider struct {
	priority int
	name     string
	packer   PackerProviderFunc
	unpacker UnpackerProviderFunc
}

var providers []*provider

func RegisterProviderFuncs(priority int, name string, packer PackerProviderFunc, unpacker UnpackerProviderFunc) error {
	lock.Lock()
	defer lock.Unlock()
	for _, p := range providers {
		if p.name == name {
			return fmt.Errorf("provider %q already registered", name)
		}
	}
	providers = append(providers, &provider{
		priority: priority,
		name:     name,
		packer:   packer,
		unpacker: unpacker,
	})
	slices.SortFunc(providers, func(i, j *provider) int {
		r := j.priority - i.priority
		if r == 0 {
			r = strings.Compare(i.name, j.name)
		}
		return r
	})
	return nil
}

func GetPackerFor(i any) (Packer, error) {
	lock.RLock()
	defer lock.RUnlock()
	for _, p := range providers {
		packer, err := p.packer(i)
		if err == nil && packer != nil {
			return packer, nil
		}
	}

	if len(providers) == 0 {
		return nil, asn1error.NewErrorf("no packer providers registered")
	}

	return nil, asn1error.NewErrorf("no packer found for %T", i)
}

func GetUnpackerFor(i any) (Unpacker, error) {
	lock.RLock()
	defer lock.RUnlock()
	for _, p := range providers {
		unpacker, err := p.unpacker(i)
		if err == nil && unpacker != nil {
			return unpacker, nil
		}
	}
	if len(providers) == 0 {
		return nil, asn1error.NewErrorf("no packer providers registered")
	}
	return nil, asn1error.NewErrorf("no packer found for %T", i)
}
