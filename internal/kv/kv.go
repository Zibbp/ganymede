package kv

var kv *Store

type Store struct {
	db map[string]string
}

func init() {
	kv = &Store{db: make(map[string]string)}
}

func DB() *Store {
	return kv
}

//func (k Store) NewStore() *Store {
//
//	return &Store{db: make(map[string]string)}
//}

//func KV() *Store {
//	return NewStore()
//}

func (k Store) Get(key string) string {
	return k.db[key]
}

func (k Store) Set(key string, value string) {
	k.db[key] = value
}
