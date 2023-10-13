package memorypool

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testNewF struct {
	t  []*testNew
	t1 [16]int
}

type testNew struct {
	a string
	b int
}

func TestNew(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a := New[testNew](ac)
	a.a = ac.NewString("hello")
	a.b = 12
	t.Logf("%s %d\n", a.a, a.b)
	ac.Debug()
	runtime.GC()
	ac.Debug()
	assert.EqualValues(t, a.a, "hello")
	assert.EqualValues(t, a.b, 12)
	runtime.KeepAlive(ac)
}

func TestNewSlice(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	f := func() *testNewF {
		tf := New[testNewF](ac)
		a := NewSlice[*testNew](ac, 1, 1)

		a = Append[*testNew](ac, a, []*testNew{{a: "anihao", b: 12}, {a: "anihao2", b: 122}}...)
		a = Append[*testNew](ac, a, []*testNew{{a: "anihao3", b: 123}}...)
		// t.Logf("a[3]: %s %d\n", a[3].a, a[3].b)
		tf.t = a
		return tf
	}
	tf := f()

	b := NewSlice[testNew](ac, 0, 100)
	b = Append[testNew](ac, b, []testNew{{a: "bnihaob", b: 123}}...)
	// t.Logf("b: %p\n", &b)
	// t.Logf("b[0]: %s %d\n", b[0].a, b[0].b)

	runtime.GC()

	assert.EqualValues(t, "anihao", tf.t[1].a)
	assert.EqualValues(t, 12, tf.t[1].b)
	assert.EqualValues(t, "anihao2", tf.t[2].a)
	assert.EqualValues(t, 122, tf.t[2].b)
	assert.EqualValues(t, "anihao3", tf.t[3].a)
	assert.EqualValues(t, 123, tf.t[3].b)

	assert.EqualValues(t, "bnihaob", b[0].a)
	assert.EqualValues(t, 123, b[0].b)
	runtime.KeepAlive(ac)
}

func TestSliceAppend(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a := NewSlice[testNew](ac, 0, 1)
	for i := 0; i < 100_000; i++ {
		a = Append[testNew](ac, a, []testNew{{a: "nihao", b: 12}, {a: "nihao2", b: 21}}...)
	}
	t.Logf("%s %d\n", a[0].a, a[0].b)
	assert.EqualValues(t, a[0].a, "nihao")
	assert.EqualValues(t, a[0].b, 12)
	assert.EqualValues(t, a[1].a, "nihao2")
	assert.EqualValues(t, a[1].b, 21)
	assert.EqualValues(t, a[99999].a, "nihao2")
	assert.EqualValues(t, a[99999].b, 21)
	runtime.KeepAlive(ac)
}

func TestAllocPoolReuse(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a := New[testNew](ac)
	a.a = ac.NewString("hello")
	a.b = 12
	t.Logf("%s %d\n", a.a, a.b)
	assert.EqualValues(t, a.a, "hello")
	assert.EqualValues(t, a.b, 12)
	ac.ReturnAlloctorToPool()

	ac = NewAlloctorFromPool(0)
	a = New[testNew](ac)
	a.a = ac.NewString("ni")
	a.b = 1123
	t.Logf("%s %d\n", a.a, a.b)
	assert.EqualValues(t, a.a, "ni")
	assert.EqualValues(t, a.b, 1123)
	ac.ReturnAlloctorToPool()

	ac = NewAlloctorFromPool(0)
	a = New[testNew](ac)
	a.a = ac.NewString("ni")
	a.b = 1123
	t.Logf("%s %d\n", a.a, a.b)
	assert.EqualValues(t, a.a, "ni")
	assert.EqualValues(t, a.b, 1123)

	runtime.KeepAlive(ac)
}

func TestAllocPoolMerge(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a := New[testNew](ac)
	a.a = ac.NewString("hello")
	a.b = 12
	assert.EqualValues(t, a.a, "hello")
	assert.EqualValues(t, a.b, 12)

	// 新内存池1
	ac1 := NewAlloctorFromPool(ac.BlockSize())
	a1 := New[testNew](ac)
	a1.a = ac1.NewString("ni1")
	a1.b = 1123
	ac.Merge(ac1)
	runtime.KeepAlive(ac1)
	assert.EqualValues(t, a1.a, "ni1")
	assert.EqualValues(t, a1.b, 1123)

	// 新内存池2
	ac2 := NewAlloctorFromPool(ac.BlockSize())
	a2 := New[testNew](ac)
	a2.a = ac2.NewString("ni2")
	a2.b = 1123
	ac.Merge(ac2)
	runtime.KeepAlive(ac2)
	assert.EqualValues(t, a2.a, "ni2")
	assert.EqualValues(t, a2.b, 1123)

	// 合并后内存信息
	assert.EqualValues(t, len(ac.blocks), 3)
	assert.EqualValues(t, ac.bidx, 2)

	a = New[testNew](ac)
	a.a = ac.NewString("hello3")
	a.b = 12
	assert.EqualValues(t, a.a, "hello3")
	assert.EqualValues(t, a.b, 12)

	// 再次check内存信息
	assert.EqualValues(t, len(ac.blocks), 3)
	assert.EqualValues(t, ac.bidx, 2)

	runtime.KeepAlive(ac)
}
