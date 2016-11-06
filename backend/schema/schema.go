package schema

import (
	"fmt"

	"github.com/Shark/powerdns-consul/backend/store"
)

type Schema interface {
	HasZone(string) (bool, error)
	Resolve(*store.Query) ([]*store.Entry, error)
	Store() store.Store
}

func NewSchema(name string, store store.Store, defaultTTL uint32) (schema Schema, err error) {
	switch name {
	case "flat":
		return NewFlatSchema(store, defaultTTL), nil
	}

	return nil, fmt.Errorf("Unsupported schema %s", name)
}
