package memorypool

import (
	"testing"
)

type allocTest1 struct {
	A string
	B []int32
	C *allocTest2
}

type allocTest2 struct {
	D string
}

const objnum = 1000

// BenchmarkStructRawAlloc ...
func BenchmarkStructRawAlloc(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := make([]*allocTest1, 0, 1)
		for i := 0; i < objnum; i++ {
			d1 = append(d1, &allocTest1{
				A: "123123123123",
				B: append([]int32{}, []int32{1, 2, 3, 4, 5, 6, 7}...),
				C: &allocTest2{
					D: "123123123123",
				},
			})
		}
	}
}

// BenchmarkStructPoolAlloc ...
func BenchmarkStructPoolAlloc(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := NewSlice[*allocTest1](ac, 0, 1)
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = AppendMulti[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			d1 = AppendMulti[*allocTest1](ac, d1, d)
		}
		ac.Reset()
	}
}

// BenchmarkIntSliceRawAlloc ...
func BenchmarkIntSliceRawAlloc(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := make([]int, 0, objnum)
		for i := 0; i < objnum; i++ {
			d1 = append(d1, i)
		}
	}
}

// BenchmarkIntSlicePoolAllocAppendInplaceMulti ...
func BenchmarkIntSlicePoolAllocAppendInplaceMulti(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := NewSlice[int](ac, 0, objnum)
		for i := 0; i < objnum; i++ {
			d1 = AppendInplaceMulti[int](ac, d1, i)
		}
		ac.Reset()
	}
}

// BenchmarkIntSlicePoolAllocAppendInplace ...
func BenchmarkIntSlicePoolAllocAppendInplace(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := NewSlice[int](ac, 0, objnum)
		for i := 0; i < objnum; i++ {
			d1 = AppendInplace[int](ac, d1, i)
		}
		ac.Reset()
	}
}

// BenchmarkIntSlicePoolAllocAppendInbound ...
func BenchmarkIntSlicePoolAllocAppendInbound(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d1 := NewSlice[int](ac, 0, objnum)
		for i := 0; i < objnum; i++ {
			d1 = AppendInbound[int](ac, d1, i)
		}
		ac.Reset()
	}
}
