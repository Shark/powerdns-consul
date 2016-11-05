package store

type Pair interface {
	Key() string
	Value() []byte
	LastIndex() uint64
}

func NewPair(key string, value []byte, lastIndex uint64) (pair Pair) {
	return &PairImpl{key, value, lastIndex}
}

type PairImpl struct {
	key       string
	value     []byte
	lastIndex uint64
}

func (p *PairImpl) Key() string {
	return p.key
}

func (p *PairImpl) Value() []byte {
	return p.value
}

func (p *PairImpl) LastIndex() uint64 {
	return p.lastIndex
}
