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

func (S *SyncMap[K, V]) GetI(key K) (v V) {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()
	return S.MAP[key]
}

type _Tuple[K comparable, V any] struct {
	Key K
	Value V
}

func (S *SyncMap[K, V]) Iter() <-chan _Tuple[K, V] {
	S.MUTEX.Lock()
	defer S.MUTEX.Unlock()

	tchan := make(chan _Tuple[K, V], len(S.MAP))
	for k,v := range S.MAP {
		tchan<-_Tuple[K, V]{k, v}
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

