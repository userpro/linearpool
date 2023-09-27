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
	ac := NewAlloctorFromPool()
	a := New[testNew](ac)
	a.a = ac.NewString("hello")
	a.b = 12
	t.Logf("%s %d\n", a.a, a.b)
	ac.Debug()
	runtime.GC()
	ac.Debug()
	assert.EqualValues(t, a.a, "hello")
	assert.EqualValues(t, a.b, 12)
}

func TestNewSlice(t *testing.T) {
	ac := NewAlloctorFromPool()
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
}

func TestSliceAppend(t *testing.T) {
	ac := NewAlloctorFromPool()
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
}

func TestAllocPool(t *testing.T) {
	ac := NewAlloctorFromPool()
	a := New[testNew](ac)
	a.a = ac.NewString("hello")
	a.b = 12
	t.Logf("%s %d\n", a.a, a.b)
	assert.EqualValues(t, a.a, "hello")
	assert.EqualValues(t, a.b, 12)
	ReturnAlloctorToPool(ac)

	ac = NewAlloctorFromPool()
	a = New[testNew](ac)
	a.a = ac.NewString("ni")
	a.b = 1123
	t.Logf("%s %d\n", a.a, a.b)
	assert.EqualValues(t, a.a, "ni")
	assert.EqualValues(t, a.b, 1123)
}
