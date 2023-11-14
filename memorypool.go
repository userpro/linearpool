package memorypool

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

const (
	DiKB      int64 = 1 << 10
	DiMB      int64 = DiKB << 10
	DiGB      int64 = DiMB << 10
	blockSize int64 = DiKB * 4 // 4KB/block
)

var (
	SliceExtendRatio        = 2.5
	BugfixClearPointerInMem = true
	BugfixCorruptOtherMem   = true

	allocatorPool = sync.Pool{
		New: func() any {
			return &Allocator{bidx: -1}
		},
	}
)

// Allocator 分配器
type Allocator struct {
	blockSize  int64
	curBlock   *sliceHeader
	blocks     []*sliceHeader
	hugeBlocks []*sliceHeader
	bidx       int // 当前在第几个 block 进行分配

	externalPtr    []unsafe.Pointer
	externalSlice  []unsafe.Pointer
	externalString []unsafe.Pointer
	externalMap    []any
	externalFunc   []any

	subAlloctor []*Allocator // 子分配器
}

// NewAlloctorFromPool 新建分配池
func NewAlloctorFromPool(sz int64) *Allocator {
	ac := allocatorPool.Get().(*Allocator)
	if sz <= 0 {
		sz = blockSize
	}
	ac.blockSize = sz

	if ac.bidx < 0 {
		ac.newBlock()
	}
	return ac
}

func (ac *Allocator) newBlockWithSz(need int64) *sliceHeader {
	t := make([]byte, 0, need)
	b := (*sliceHeader)(unsafe.Pointer(&t))
	ac.hugeBlocks = append(ac.hugeBlocks, b)
	return b
}

func (ac *Allocator) newBlock() *sliceHeader {
	ac.bidx++
	// 可能复用之前的blocks
	if len(ac.blocks) > ac.bidx {
		b := ac.blocks[ac.bidx]
		ac.curBlock = b
		return b
	}

	t := make([]byte, 0, ac.blockSize)
	b := (*sliceHeader)(unsafe.Pointer(&t))
	ac.curBlock = b
	ac.blocks = append(ac.blocks, b)
	return b
}

func (ac *Allocator) alloc(need int64) unsafe.Pointer {
	if need == 0 && BugfixCorruptOtherMem {
		return nil
	}

	// round up
	needAligned := need
	if need%ptrSize != 0 {
		needAligned = (need + ptrSize + 1) & ^(ptrSize - 1)
	}

	// 分配小型对象
	if ac.blockSize >= needAligned {
		b := ac.curBlock
		if b.Len+int64(needAligned) > b.Cap {
			b = ac.newBlock()
		}

		ptr := unsafe.Add(b.Data, b.Len)
		b.Len += needAligned
		// fmt.Printf("bidx: %d, blocksize: %d, alloc need: %d, needAligned: %d, len: %d, %v - %v\n",
		// 	ac.bidx, len(ac.blocks), need, needAligned, b.Len, ptr, unsafe.Add(b.Data, b.Cap-1))
		return ptr
	}

	// 分配巨型对象
	b := ac.newBlockWithSz(needAligned)
	ptr := b.Data
	b.Len = b.Cap
	// fmt.Printf("huge alloc need: %d, needAligned: %d, cap: %d, %v - %v\n",
	// 	need, needAligned, b.Cap, b.Data, unsafe.Add(b.Data, b.Cap-1))
	return ptr
}

// Reset 重置内存信息
func (ac *Allocator) Reset() {
	ac.bidx = 0
	ac.curBlock = ac.blocks[0]
	for _, b := range ac.blocks {
		if b.Len > 0 {
			memclrNoHeapPointers(b.Data, uintptr(b.Len))
			b.Len = 0
		}
	}
	ac.blocks = ac.blocks[:1]
	ac.hugeBlocks = nil // 大对象直接释放 避免过多占用内存

	ac.externalPtr = ac.externalPtr[:0]
	ac.externalSlice = ac.externalSlice[:0]
	ac.externalString = ac.externalString[:0]
	ac.externalMap = ac.externalMap[:0]
	ac.externalFunc = ac.externalFunc[:0]

	for i, subAc := range ac.subAlloctor {
		subAc.Reset()
		ac.subAlloctor[i] = nil
	}
	ac.subAlloctor = ac.subAlloctor[:0]
}

// ReturnAlloctorToPool 归还分配池
func (ac *Allocator) ReturnAlloctorToPool() {
	ac.Reset()
	allocatorPool.Put(ac)
}

// BlockSize 获取当前内存池的 blocksize
func (ac *Allocator) BlockSize() int64 {
	return ac.blockSize
}

// AddSubAlloctor 新增子分配器
func (ac *Allocator) AddSubAlloctor(sub *Allocator) {
	ac.subAlloctor = append(ac.subAlloctor, sub)
}

// SubAlloctor 获取子分配器
func (ac *Allocator) SubAlloctor() []*Allocator {
	return ac.subAlloctor
}

// Merge 合并其他内存池
func (ac *Allocator) Merge(src *Allocator) *Allocator {
	ac.blocks = append(ac.blocks, src.blocks[:src.bidx+1]...)
	ac.hugeBlocks = append(ac.hugeBlocks, src.hugeBlocks...)
	ac.bidx = ac.bidx + src.bidx + 1

	ac.externalPtr = append(ac.externalPtr, src.externalPtr...)
	ac.externalSlice = append(ac.externalSlice, src.externalSlice...)
	ac.externalString = append(ac.externalString, src.externalString...)
	ac.externalMap = append(ac.externalMap, src.externalMap...)
	ac.externalFunc = append(ac.externalFunc, src.externalFunc...)
	return ac
}

// KeepAlive GC保活
func (ac *Allocator) KeepAlive(ptr interface{}) {
	d := data(ptr)
	if d == nil {
		return
	}

	k := reflect.TypeOf(ptr).Kind()
	switch k {
	case reflect.Ptr:
		ac.externalPtr = append(ac.externalPtr, d)
	case reflect.Slice:
		ac.externalSlice = append(ac.externalSlice, (*sliceHeader)(d).Data)
	case reflect.String:
		ac.externalString = append(ac.externalString, (*stringHeader)(d).Data)
	case reflect.Map:
		ac.externalMap = append(ac.externalMap, d)
	case reflect.Func:
		ac.externalFunc = append(ac.externalFunc, ptr)
	default:
		panic(fmt.Errorf("unsupported type: %v", k))
	}
}

// New 分配新对象
func New[T any](ac *Allocator) (r *T) {
	r = (*T)(ac.alloc(int64(unsafe.Sizeof(*r))))
	return r
}

// NewSlice does not zero the slice automatically, this is OK with most cases and can improve the performance.
// zero it yourself for your need.
func NewSlice[T any](ac *Allocator, len, cap int) (r []T) {
	// keep same with systems `new`.
	if len > cap {
		panic("NewSlice: cap out of range")
	}

	if BugfixCorruptOtherMem && cap == 0 {
	}

	slice := (*sliceHeader)(unsafe.Pointer(&r))
	var t T
	slice.Data = ac.alloc(int64(cap) * int64(unsafe.Sizeof(t)))
	slice.Len = int64(len)
	slice.Cap = int64(cap)
	return r
}

// AppendMulti append slice
func AppendMulti[T any](ac *Allocator, s []T, elems ...T) []T {
	if len(elems) == 0 {
		return s
	}

	h := (*sliceHeader)(unsafe.Pointer(&s))
	elemSz := int(unsafe.Sizeof(elems[0]))
	src := (*sliceHeader)(unsafe.Pointer(&elems))

	// grow
	if h.Len+src.Len > h.Cap {
		pre := *h
		h.Cap = int64(roundupsize(uintptr(pre.Cap + int64(len(elems)))))
		sz := int(h.Cap) * elemSz
		h.Data = ac.alloc(int64(sz))
		memmoveNoHeapPointers(h.Data, pre.Data, uintptr(int(pre.Len)*elemSz))
	}

	// append
	memmoveNoHeapPointers(unsafe.Add(h.Data, elemSz*int(h.Len)), src.Data, uintptr(elemSz*int(src.Len)))
	h.Len += src.Len

	return s
}

// Append append slice
func Append[T any](ac *Allocator, s []T, elem T) []T {
	h := (*sliceHeader)(unsafe.Pointer(&s))
	elemSz := int(unsafe.Sizeof(elem))

	// grow
	if h.Len+1 > h.Cap {
		pre := *h
		h.Cap = int64(roundupsize(uintptr(pre.Cap + 1)))
		sz := int(h.Cap) * elemSz
		h.Data = ac.alloc(int64(sz))
		memmoveNoHeapPointers(h.Data, pre.Data, uintptr(int(pre.Len)*elemSz))
	}

	// append
	*(*T)(unsafe.Add(h.Data, elemSz*int(h.Len))) = elem
	h.Len += 1

	return s
}

// AppendInplaceMulti 与 NewSlice 之间没有任何新的内存分配时使用
func AppendInplaceMulti[T any](ac *Allocator, s []T, elems ...T) []T {
	if len(elems) == 0 {
		return s
	}

	h := (*sliceHeader)(unsafe.Pointer(&s))
	elemSz := int(unsafe.Sizeof(elems[0]))
	src := (*sliceHeader)(unsafe.Pointer(&elems))

	// grow
	if h.Len+src.Len > h.Cap {
		pre := *h
		newcap := int64(roundupsize(uintptr(pre.Cap + int64(len(elems)))))
		curB := ac.curBlock
		growthInplace := (newcap - pre.Cap) * int64(elemSz)
		if curB.Len+growthInplace < curB.Cap {
			curB.Len += growthInplace
		} else {
			sz := int(newcap) * elemSz
			h.Data = ac.alloc(int64(sz))
			memmoveNoHeapPointers(h.Data, pre.Data, uintptr(int(pre.Len)*elemSz))
		}
		h.Cap = newcap
	}

	// append
	memmoveNoHeapPointers(unsafe.Add(h.Data, elemSz*int(h.Len)), src.Data, uintptr(elemSz*int(src.Len)))
	h.Len += src.Len

	return s
}

// AppendInplace 与 NewSlice 之间没有任何新的内存分配时使用
func AppendInplace[T any](ac *Allocator, s []T, elem T) []T {
	h := (*sliceHeader)(unsafe.Pointer(&s))
	elemSz := int(unsafe.Sizeof(elem))

	// grow
	if h.Len+1 > h.Cap {
		pre := *h
		newcap := int64(roundupsize(uintptr(pre.Cap + 1)))
		if ac.curBlock.Len+newcap-h.Cap < ac.curBlock.Cap {
			ac.curBlock.Len += newcap - h.Cap
		} else {
			sz := int(newcap) * elemSz
			h.Data = ac.alloc(int64(sz))
			memmoveNoHeapPointers(h.Data, pre.Data, uintptr(int(pre.Len)*elemSz))
		}
		h.Cap = newcap
	}

	// append
	*(*T)(unsafe.Add(h.Data, elemSz*int(h.Len))) = elem
	h.Len += 1

	return s
}

// AppendInbound 确定范围内 append
func AppendInbound[T any](ac *Allocator, s []T, elem T) []T {
	if len(s)+1 > cap(s) {
		panic(fmt.Sprintf("index %d out of bound cap %d", len(s), cap(s)))
	}

	return append(s, elem)
}

// NewString 从内存池分配 string
func (ac *Allocator) NewString(v string) string {
	if len(v) == 0 {
		return ""
	}
	h := (*stringHeader)(unsafe.Pointer(&v))
	ptr := ac.alloc(int64(h.Len))
	if ptr != nil {
		memmoveNoHeapPointers(ptr, h.Data, uintptr(h.Len))
	}
	h.Data = ptr
	return v
}

// Debug 输出 debug 信息
func (ac *Allocator) Debug() {
	fmt.Printf("\n* bidx: %d\n", ac.bidx)
	fmt.Printf("* blocks: \n")
	fmt.Printf(" - curblock: len(%d) cap(%d) addr[%p - %p]\n", ac.curBlock.Len, ac.curBlock.Cap, ac.curBlock.Data, unsafe.Add(ac.curBlock.Data, ac.curBlock.Cap-1))
	for i, b := range ac.blocks {
		b1 := *(*[]byte)(unsafe.Pointer(b))
		fmt.Printf(" - b[%d]: len(%d) cap(%d) addr[%p - %p] data: %v\n", i, b.Len, b.Cap, b.Data, unsafe.Add(b.Data, b.Cap-1), b1)
	}

	if len(ac.hugeBlocks) > 0 {
		fmt.Printf("* huge blocks: \n")
		for i, b := range ac.hugeBlocks {
			b1 := *(*[]byte)(unsafe.Pointer(b))
			fmt.Printf(" - hb[%d]: len(%d) cap(%d) addr[%p - %p] data: %v\n", i, b.Len, b.Cap, b.Data, unsafe.Add(b.Data, b.Cap-1), b1)
		}
	}
	fmt.Printf("\n")
}

//============================================================================
// Protobuf2 APIs
//============================================================================

// Bool ...
func (ac *Allocator) Bool(v bool) (r *bool) {
	r = (*bool)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Int ...
func (ac *Allocator) Int(v int) (r *int) {
	r = (*int)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Int32 ...
func (ac *Allocator) Int32(v int32) (r *int32) {
	r = (*int32)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Uint32 ...
func (ac *Allocator) Uint32(v uint32) (r *uint32) {
	r = (*uint32)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Int64 ...
func (ac *Allocator) Int64(v int64) (r *int64) {
	r = (*int64)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Uint64 ...
func (ac *Allocator) Uint64(v uint64) (r *uint64) {
	r = (*uint64)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Float32 ...
func (ac *Allocator) Float32(v float32) (r *float32) {
	r = (*float32)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// Float64 ...
func (ac *Allocator) Float64(v float64) (r *float64) {
	r = (*float64)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = v
	return
}

// String ...
func (ac *Allocator) String(v string) (r *string) {
	r = (*string)(ac.alloc(int64(unsafe.Sizeof(v))))
	*r = ac.NewString(v)
	return
}
