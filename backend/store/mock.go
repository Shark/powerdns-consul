package store

type MockStore struct {
	GetFunc       func(string) (Pair, error)
	PutFunc       func(key string, value []byte, options *WriteOptions) error
	ListFunc      func(directory string) ([]Pair, error)
	AtomicPutFunc func(key string, value []byte, previous Pair, options *WriteOptions) (bool, Pair, error)
}

func (kv MockStore) Get(key string) (Pair, error) {
	return kv.GetFunc(key)
}

func (kv MockStore) Put(key string, value []byte, options *WriteOptions) error {
	return kv.PutFunc(key, value, options)
}

func (kv MockStore) List(directory string) ([]Pair, error) {
	return kv.ListFunc(directory)
}

func (kv MockStore) AtomicPut(key string, value []byte, previous Pair, options *WriteOptions) (bool, Pair, error) {
	return kv.AtomicPutFunc(key, value, previous, options)
}
