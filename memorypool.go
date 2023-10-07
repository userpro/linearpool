package memorypool

import (
	"fmt"
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
	// our memory is much cheaper than systems,
	// so we can be more aggressive than `append`.
	SliceExtendRatio = 2.5

	BugfixClearPointerInMem = true
	BugfixCorruptOtherMem   = true

	allocatorPool = sync.Pool{
		New: func() any {
			ac := &Allocator{bidx: -1}
			ac.newBlock()
			return ac
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
}

// NewAlloctorFromPool 新建分配池
func NewAlloctorFromPool(sz int64) *Allocator {
	ac := allocatorPool.Get().(*Allocator)
	if sz <= 0 {
		sz = blockSize
	}
	ac.blockSize = sz
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

func (ac *Allocator) reset() {
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
			ac.bidx++
			if len(ac.blocks) <= ac.bidx {
				b = ac.newBlock()
			}
		}

		ptr := unsafe.Add(b.Data, b.Len)
		b.Len += needAligned
		// fmt.Printf("bidx: %d, zero: %v, alloc need: %d, needAligned: %d, len: %d, %v - %v\n",
		// 	ac.bidx, zero, need, needAligned, b.Len, ptr, unsafe.Add(b.Data, b.Cap-1))
		return ptr
	}

	// 分配巨型对象
	b := ac.newBlockWithSz(needAligned)
	ptr := b.Data
	b.Len = b.Cap
	// fmt.Printf("huge alloc zero: %v, need: %d, needAligned: %d, cap: %d, %v - %v\n",
	// 	zero, need, needAligned, b.Cap, b.Data, unsafe.Add(b.Data, b.Cap-1))
	return ptr
}

// ReturnAlloctorToPool 归还分配池
func (ac *Allocator) ReturnAlloctorToPool() {
	ac.reset()
	allocatorPool.Put(ac)
}

// New 分配新对象
func New[T any](ac *Allocator) (r *T) {
	if ac == nil {
		return new(T)
	}

	r = (*T)(ac.alloc(int64(unsafe.Sizeof(*r))))
	return r
}

// NewSlice does not zero the slice automatically, this is OK with most cases and can improve the performance.
// zero it yourself for your need.
func NewSlice[T any](ac *Allocator, len, cap int) (r []T) {
	if ac == nil {
		return make([]T, len, cap)
	}

	// keep same with systems `new`.
	if len > cap {
		panic("NewSlice: cap out of range")
	}

	if BugfixCorruptOtherMem {
		if cap == 0 {
			return nil
		}
	}

	slice := (*sliceHeader)(unsafe.Pointer(&r))
	var t T
	// fmt.Println(cap, unsafe.Sizeof(t2))
	slice.Data = ac.alloc(int64(cap) * int64(unsafe.Sizeof(t)))
	slice.Len = int64(len)
	slice.Cap = int64(cap)
	return r
}

func Append[T any](ac *Allocator, s []T, elems ...T) []T {
	if ac == nil {
		return append(s, elems...)
	}

	if len(elems) == 0 {
		return s
	}

	h := (*sliceHeader)(unsafe.Pointer(&s))
	elemSz := int(unsafe.Sizeof(elems[0]))

	// grow
	if h.Len >= h.Cap {
		pre := *h

		cur := float64(h.Cap)
		h.Cap = max(int64(cur*SliceExtendRatio), pre.Cap+int64(len(elems)))
		if h.Cap < 16 {
			h.Cap = 16
		}

		sz := int(h.Cap) * elemSz
		h.Data = ac.alloc(int64(sz))
		memmoveNoHeapPointers(h.Data, pre.Data, uintptr(int(pre.Len)*elemSz))
	}

	// append
	src := (*sliceHeader)(unsafe.Pointer(&elems))
	memmoveNoHeapPointers(unsafe.Add(h.Data, elemSz*int(h.Len)), src.Data, uintptr(elemSz*int(src.Len)))
	h.Len += src.Len

	return s
}

func (ac *Allocator) NewString(v string) string {
	if ac == nil {
		return v
	}
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

func (ac *Allocator) Bool(v bool) (r *bool) {
	if ac == nil {
		r = new(bool)
	} else {
		r = (*bool)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Int(v int) (r *int) {
	if ac == nil {
		r = new(int)
	} else {
		r = (*int)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Int32(v int32) (r *int32) {
	if ac == nil {
		r = new(int32)
	} else {
		r = (*int32)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Uint32(v uint32) (r *uint32) {
	if ac == nil {
		r = new(uint32)
	} else {
		r = (*uint32)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Int64(v int64) (r *int64) {
	if ac == nil {
		r = new(int64)
	} else {
		r = (*int64)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Uint64(v uint64) (r *uint64) {
	if ac == nil {
		r = new(uint64)
	} else {
		r = (*uint64)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Float32(v float32) (r *float32) {
	if ac == nil {
		r = new(float32)
	} else {
		r = (*float32)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) Float64(v float64) (r *float64) {
	if ac == nil {
		r = new(float64)
	} else {
		r = (*float64)(ac.alloc(int64(unsafe.Sizeof(v))))
	}
	*r = v
	return
}

func (ac *Allocator) String(v string) (r *string) {
	if ac == nil {
		r = new(string)
		*r = v
	} else {
		// FIX: invalid pointer in the allocated memory may cause panic in the write barrier.
		const zero = true

		r = (*string)(ac.alloc(int64(unsafe.Sizeof(v))))
		*r = ac.NewString(v)
	}
	return
}
