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
	d1 := make([]*allocTest1, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d1 = append(d1, &allocTest1{
				A: "123123123123",
				B: append([]int32{}, []int32{1, 2, 3, 4, 5, 6, 7}...),
				C: &allocTest2{
					D: "123123123123",
				},
			})
		}
		d1 = d1[:0]
	}
}

// BenchmarkStructPoolAlloc ...
func BenchmarkStructPoolAlloc(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	d1 := NewSlice[*allocTest1](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = AppendMulti[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			d1 = AppendMulti[*allocTest1](ac, d1, d)
		}
		d1 = d1[:0]
	}
}

// BenchmarkStructPoolAllocOne ...
func BenchmarkStructPoolAllocOne(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	d1 := NewSlice[*allocTest1](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = AppendMulti[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			d1 = Append[*allocTest1](ac, d1, d)
		}
		d1 = d1[:0]
	}
}

// BenchmarkIntSliceRawAlloc ...
func BenchmarkIntSliceRawAlloc(b *testing.B) {
	d1 := make([]int, 0, objnum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d1 = append(d1, i)
		}
		d1 = d1[:0]
	}
}

// BenchmarkIntSlicePoolAllocAppendInplaceMulti ...
func BenchmarkIntSlicePoolAllocAppendInplaceMulti(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	d1 := NewSlice[int](ac, 0, objnum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d1 = AppendInplaceMulti[int](ac, d1, i)
		}
		d1 = d1[:0]
	}
}

// BenchmarkIntSlicePoolAllocAppendInplace ...
func BenchmarkIntSlicePoolAllocAppendInplace(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	d1 := NewSlice[int](ac, 0, objnum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d1 = AppendInplace[int](ac, d1, i)
		}
		d1 = d1[:0]
	}
}

// BenchmarkIntSlicePoolAllocAppendInbound ...
func BenchmarkIntSlicePoolAllocAppendInbound(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	d1 := NewSlice[int](ac, 0, objnum)
	// d2 := d1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d1 = AppendInbound[int](ac, d1, i)
		}
		d1 = d1[:0]
	}

	// h1 := (*sliceHeader)(unsafe.Pointer(&d1))
	// h2 := (*sliceHeader)(unsafe.Pointer(&d2))
	// assert.EqualValues(b, h1.Data, h2.Data)
}
