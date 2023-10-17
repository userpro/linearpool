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

// BenchmarkRawAlloc ...
func BenchmarkRawAlloc(b *testing.B) {
	data := make([]*allocTest1, 0, objnum)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			data = append(data, &allocTest1{
				A: "123123123123",
				B: []int32{1, 2, 3, 4, 5, 6, 7},
				C: &allocTest2{
					D: "123123123123",
				},
			})
		}
		data = data[:0]
	}
}

// BenchmarkPoolAlloc ...
func BenchmarkPoolAlloc(b *testing.B) {
	data := make([]*allocTest1, 0, objnum)
	ac := NewAlloctorFromPool(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for i := 0; i < objnum; i++ {
			d := New[allocTest1](ac)
			d.A = ac.NewString("123123123123")
			d.B = NewSlice[int32](ac, 0, 7)
			d.B = Append[int32](ac, d.B, []int32{1, 2, 3, 4, 5, 6, 7}...)
			d.C = New[allocTest2](ac)
			d.C.D = ac.NewString("123123123123")
			data = Append[*allocTest1](ac, data, d)
		}
		data = data[:0]
	}
}
