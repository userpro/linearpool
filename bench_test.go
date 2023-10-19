package memorypool

import "testing"

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
	data := make([]*allocTest1, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			data = append(data, &allocTest1{
				A: "123123123123",
				B: append([]int32{}, []int32{1, 2, 3, 4, 5, 6, 7}...),
				C: &allocTest2{
					D: "123123123123",
				},
			})
		}
		data = data[:0]
	}
}

// BenchmarkStructPoolAlloc ...
func BenchmarkStructPoolAlloc(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	data := NewSlice[*allocTest1](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = AppendMulti[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			data = AppendMulti[*allocTest1](ac, data, d)
		}
		data = data[:0]
	}
}

// BenchmarkStructPoolAllocOne ...
func BenchmarkStructPoolAllocOne(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	data := NewSlice[*allocTest1](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = AppendMulti[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			data = Append[*allocTest1](ac, data, d)
		}
		data = data[:0]
	}
}

// BenchmarkIntSliceRawAlloc ...
func BenchmarkIntSliceRawAlloc(b *testing.B) {
	data := make([]int, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			data = append(data, i)
		}
		data = data[:0]
	}
}

// BenchmarkIntSlicePoolAlloc ...
func BenchmarkIntSlicePoolAlloc(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	data := NewSlice[int](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			data = AppendInplaceMulti[int](ac, data, i)
		}
		data = data[:0]
	}
}

// BenchmarkIntSlicePoolAllocOne ...
func BenchmarkIntSlicePoolAllocOne(b *testing.B) {
	ac := NewAlloctorFromPool(DiMB)
	data := NewSlice[int](ac, 0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			data = AppendInplace[int](ac, data, i)
		}
		data = data[:0]
	}
}
