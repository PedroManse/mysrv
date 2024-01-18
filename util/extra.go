package util

import (
	"hash/fnv"
	"sync"
	"strings"
	"fmt"
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

func (S *SyncMap[K, V]) Lock() {
	S.MUTEX.Lock()
}

func (S *SyncMap[K, V]) Unlock() {
	S.MUTEX.Unlock()
}

func NewSyncMap[Key comparable, Value any]() *SyncMap[Key, Value] {
	return &SyncMap[Key, Value]{
		make(map[Key]Value),
		sync.Mutex{},
	}
}

func MakeSyncMap[Key comparable, Value any]() SyncMap[Key, Value] {
	return SyncMap[Key, Value]{
		make(map[Key]Value),
		sync.Mutex{},
	}
}

func ISyncMap[K comparable, V any](mp map[K]V) (*SyncMap[K, V]) {
	return &SyncMap[K, V]{
		MAP: mp,
		MUTEX: sync.Mutex{},
	}
}

func (S *SyncMap[K, V]) Init() {
	S.MAP = make(map[K]V)
	S.MUTEX = sync.Mutex{}
}

func (S *SyncMap[K, V]) Set(key K, value V) {
	S.Lock()
	defer S.Unlock()
	S.MAP[key] = value
}

func (S *SyncMap[K, V]) Get(key K) (v V, has bool) {
	S.Lock()
	defer S.Unlock()
	v, has = S.MAP[key]
	return
}

func (S *SyncMap[K, V]) Unset(key K) {
	S.Lock()
	defer S.Unlock()
	delete(S.MAP, key)
}

func (S *SyncMap[K, V]) Has(key K) ( has bool ) {
	S.Lock()
	defer S.Unlock()
	_, has = S.MAP[key]
	return has
}

func (S *SyncMap[K, V]) GetI(key K) (v V) {
	S.Lock()
	defer S.Unlock()
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
	S.Lock()
	defer S.Unlock()

	tchan := make(chan Tuple[K, V], len(S.MAP))
	for k,v := range S.MAP {
		tchan<-Tuple[K, V]{k, v}
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) IterValues() <-chan V {
	S.Lock()
	defer S.Unlock()

	tchan := make(chan V, len(S.MAP))
	for _,v := range S.MAP {
		tchan<-v
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) IterKeys() <-chan K {
	S.Lock()
	defer S.Unlock()

	tchan := make(chan K, len(S.MAP))
	for k := range S.MAP {
		tchan<-k
	}
	close(tchan)
	return tchan
}

func (S *SyncMap[K, V]) Copy() (m SyncMap[K, V]) {
	S.Lock()
	defer S.Unlock()
	m.MAP = make(map[K]V)
	for k,v:=range S.MAP {
		m.MAP[k] = v
	}
	m.MUTEX = sync.Mutex{}
	return
}

func (S *SyncMap[K, V]) AMap() (m map[K]V) {
	S.Lock()
	defer S.Unlock()
	m = make(map[K]V)
	for k,v:=range S.MAP {
		m[k] = v
	}
	return
}

func (S *SyncMap[K, V]) Len() (int) {
	S.Lock()
	defer S.Unlock()
	return len(S.MAP)
}

func RevertMap[K comparable, V comparable](mp map[K]V) (newmp map[V]K) {
	newmp = make(map[V]K)
	for k,v:=range mp {
		newmp[v] = k
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

type SIntType  interface{int | int8 | int16 | int32 | int64}
type UIntType  interface{uint | uint8 | uint16 | uint32 | uint64}
type IntType  interface{SIntType | UIntType}
type FloatType  interface{float32 | float64}
type NumberType  interface{IntType | FloatType}

func Min[V NumberType](a, b V) (V) {
	if (a < b) {return a}
	return b
}

func Max[V NumberType](a, b V) (V) {
	if (a > b) {return a}
	return b
}

// implements io.Writer
type WriteBuffer struct {
	Buffer **[]byte
}

func (WB *WriteBuffer) Init() {
	buffer := &[]byte{}
	WB.Buffer = &(buffer)
}

func (WB WriteBuffer) Write(p []byte) (n int, err error) {
	nbuff := append(**WB.Buffer, p...)
	*WB.Buffer = &nbuff
	return len(p), nil
}

func (WB WriteBuffer) String() (string) {
	return string(**WB.Buffer)
}

func (WB WriteBuffer) Bytes() ([]byte) {
	return **WB.Buffer
}

func RemoveSpace(in string) (out string) {
	return strings.TrimSpace(in)
}

type ConstError string
func (err ConstError) Error() string {
	return string(err)
}

func (err ConstError) Is(target error) bool {
	return err == target
}

type DynError struct {
	Format error
	Value any
}

func (err DynError) Error() string {
	return fmt.Sprintf("%s: %v", err.Format.Error(), err.Value)
}

func (err DynError) Is(target error) bool {
	ts := target.Error()
	return ts == err.Format.Error()
}

//func MakeCacheMap[K comparable, V any](mp map[K]V, copies int) (cm CacheMap[K, V]) {
//	cm = CacheMap[K, V]{
//		mp, sync.Mutex{}, copies,
//		make([]map[K]V, copies),
//		make([]sync.Mutex, copies),
//	}
//	for i:=0;i<cm.copies;i++ {
//		cm.cps[i] = make(map[K]V)
//	}
//	for k,v := range mp {
//		for i:=0;i<cm.copies;i++ {
//			cm.cps[i][k]=v
//		}
//	}
//	return
//}
//
//type CacheMap[Key comparable, Value any] struct {
//	Origin map[Key]Value
//	mutex sync.Mutex
//	copies int
//	cps []map[Key]Value
//	lks []sync.Mutex
//}
//
//func (CM *CacheMap[K, V]) Write(key K, value V) () {
//	CM.mutex.Lock()
//	CM.Origin[key]=value
//	CM.mutex.Unlock()
//}
//
//// possibly implement cond_t with event argument,
//// to enable the reader to jump to the open map
//func (CM *CacheMap[K, V]) Read(key K, rid int) (V, bool) {
//	fmt.Printf("[%d] Reading for %v\n", rid, key)
//	j := 0
//	for {
//		fmt.Printf("[%d] %dth pass\n", rid, j)
//		j++
//		for i:=0;i<CM.copies;i++ {
//			owner := CM.lks[i].TryLock()
//			if (!owner) {continue}
//			fmt.Printf("[%d] own %d\n", rid, i)
//			time.Sleep(50000)
//			v,ok := CM.cps[i][key]
//			CM.lks[i].Unlock()
//			fmt.Printf("[%d] sell %d\n", rid, i)
//			return v, ok
//		}
//	}
//}

