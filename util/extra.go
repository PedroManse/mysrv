package util

import (
	"hash/fnv"
	"sync"
)

type HashResult = uint32
const HashBitLen = 32
func Hash(s string) HashResult {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()+uint32(90749*len(s))
}

func NewSyncMap[Key comparable, Value any]() SyncMap[Key, Value] {
	return SyncMap[Key, Value]{
		make(map[Key]Value),
		sync.Mutex{},
	}
}

type SyncMap[Key comparable, Value any] struct {
	MAP map[Key]Value
	MUTEX sync.Mutex
}

func (S *SyncMap[K, V]) Init() {
	S.MAP = make(map[K]V)
	S.MUTEX = sync.Mutex{}
}

func (S *SyncMap[K, V]) Set(key K, value V) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	S.MAP[key] = value
}

func (S *SyncMap[K, V]) Get(key K) (v V, has bool) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	v, has = S.MAP[key]
	return
}

func (S *SyncMap[K, V]) Unset(key K) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	delete(S.MAP, key)
}

func (S *SyncMap[K, V]) Has(key K) ( has bool ) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	_, has = S.MAP[key]
	return has
}

func (S *SyncMap[K, V]) GetI(key K) (v V) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	return S.MAP[key]
}

type Tuple[K any, V any] struct {
	Left K
	Right V
}

func (T Tuple[K, V]) Unpack() (K, V) {
	return T.Left, T.Right
}

func (S *SyncMap[K, V]) Iter() <-chan Tuple[K, V] {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()

	tchan := make(chan Tuple[K, V], len(S.MAP))
	for k,v := range S.MAP {
		tchan<-Tuple[K, V]{k, v}
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) IterValues() <-chan V {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()

	tchan := make(chan V, len(S.MAP))
	for _,v := range S.MAP {
		tchan<-v
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) IterKeys() <-chan K {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()

	tchan := make(chan K, len(S.MAP))
	for k := range S.MAP {
		tchan<-k
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) Copy() (m SyncMap[K, V]) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	m.MAP = make(map[K]V)
	for k,v:=range S.MAP {
		m.MAP[k] = v
	}
	m.MUTEX = sync.Mutex{}
	return
}

func (S *SyncMap[K, V]) AMap() (m map[K]V) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	m = make(map[K]V)
	for k,v:=range S.MAP {
		m[k] = v
	}
	return
}

type listener[T any] func(T) (suicide bool)
type Event[T any] []listener[T]

func (E *Event[T]) Listen(l listener[T]) {
	*E = append(*E, l)
}

func (E *Event[T]) Alert(value T) {
	for i, handler := range *E {
		if (handler(value)) {
			*E = append((*E)[:i], (*E)[i+1:]...)
		}
	}
}

