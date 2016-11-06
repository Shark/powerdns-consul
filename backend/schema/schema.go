package schema

import "github.com/Shark/powerdns-consul/backend/store"

type Schema interface {
	HasZone(string) (bool, error)
	Resolve(*store.Query) ([]*store.Entry, error)
}
