package memorypool

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
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
	tmpInt := rand.Int31()
	f := func(tmpstr []byte) *testNewF {
		tf := New[testNewF](ac)
		a := NewSlice[*testNew](ac, 1, 1)

		t1 := New[testNew](ac)
		t1.a = "anihao" // 常量字符串可以这么操作
		t1.b = 12

		t2 := New[testNew](ac)
		t2.a = "anihao2" + string(tmpstr)
		t2.b = 122

		a = AppendMulti[*testNew](ac, a, t1, t2)
		tf.t = a
		return tf
	}
	tf := f([]byte(strconv.Itoa(int(tmpInt))))

	b := NewSlice[testNew](ac, 0, 100)
	b = AppendMulti[testNew](ac, b, []testNew{{a: "bnihaob", b: 123}}...)

	runtime.GC()

	assert.EqualValues(t, "anihao", tf.t[1].a)
	assert.EqualValues(t, 12, tf.t[1].b)
	assert.EqualValues(t, "anihao2"+strconv.Itoa(int(tmpInt)), tf.t[2].a)
	assert.EqualValues(t, 122, tf.t[2].b)

	assert.EqualValues(t, "bnihaob", b[0].a)
	assert.EqualValues(t, 123, b[0].b)
	runtime.KeepAlive(ac)
}

func TestSliceAppend(t *testing.T) {
	maxn := 5_00_000
	ac := NewAlloctorFromPool(0)
	a := NewSlice[testNew](ac, 0, 1)
	for i := 0; i < maxn; i++ {
		a = AppendMulti[testNew](ac, a, []testNew{{a: "nihao", b: 12}, {a: "nihao2", b: 21}}...)
	}
	t.Logf("%s %d\n", a[0].a, a[0].b)
	t.Logf("[first] bidx: %d, blocks: %d\n", ac.bidx, len(ac.blocks))
	assert.EqualValues(t, len(a), maxn*2)

	for i := 0; i < maxn; i++ {
		if i%2 != 0 {
			assert.EqualValues(t, a[i].a, "nihao2")
			assert.EqualValues(t, a[i].b, 21)
		} else {
			assert.EqualValues(t, a[i].a, "nihao")
			assert.EqualValues(t, a[i].b, 12)
		}
	}

	ac.ReturnAlloctorToPool()

	ac = NewAlloctorFromPool(0)
	a = NewSlice[testNew](ac, 0, 1)
	for i := 0; i < maxn; i++ {
		a = AppendMulti[testNew](ac, a, []testNew{{a: "nihao", b: 12}, {a: "nihao2", b: 21}}...)
	}
	t.Logf("[second] bidx: %d, blocks: %d\n", ac.bidx, len(ac.blocks))
	runtime.KeepAlive(ac)
}

// TestSliceAppend1 交错slice append
func TestSliceAppend1(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a1 := NewSlice[int](ac, 0, 1)
	a1 = AppendMulti[int](ac, a1, []int{1, 2}...) // 扩容
	a2 := NewSlice[int](ac, 0, 2)
	a2 = AppendMulti[int](ac, a2, []int{3, 4}...) // 不扩容
	a3 := NewSlice[int](ac, 0, 1)
	a3 = AppendMulti[int](ac, a3, []int{5, 6}...) // 扩容

	assert.EqualValues(t, 1, a1[0])
	assert.EqualValues(t, 2, a1[1])
	assert.EqualValues(t, 3, a2[0])
	assert.EqualValues(t, 4, a2[1])
	assert.EqualValues(t, 5, a3[0])
	assert.EqualValues(t, 6, a3[1])

	runtime.KeepAlive(ac)
}

// TestSliceAppend2 slice 扩容机制
func TestSliceAppend2(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a1 := NewSlice[int](ac, 0, 1)
	for i := 0; i < 10_000; i++ {
		a1 = AppendMulti[int](ac, a1, []int{1, 2}...) // 扩容
		// t.Log(len(a1), cap(a1))
		assert.EqualValues(t, roundupsize(uintptr(len(a1))), cap(a1))
	}
	for i := 0; i < 10_000; i++ {
		if i&1 == 0 {
			assert.EqualValues(t, 1, a1[i])
		} else {
			assert.EqualValues(t, 2, a1[i])
		}
	}

	runtime.KeepAlive(ac)
}

// TestSliceAppendInplace3 slice 原地扩容机制
func TestSliceAppendInplace3(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a1 := NewSlice[int](ac, 0, 1)
	for i := 0; i < 100_000; i++ {
		a1 = AppendInplaceMulti[int](ac, a1, []int{1, 2}...) // 扩容
		assert.EqualValues(t, roundupsize(uintptr(len(a1))), cap(a1))
	}
	for i := 0; i < 100_000; i++ {
		if i&1 == 0 {
			assert.EqualValues(t, 1, a1[i])
		} else {
			assert.EqualValues(t, 2, a1[i])
		}
	}

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

type allocKeepTest1 struct {
	ac *Allocator
	b  string
}

func TestKeepAlivePool(t *testing.T) {
	ac := NewAlloctorFromPool(0)
	a := New[allocKeepTest1](ac)
	a.ac = NewAlloctorFromPool(0)
	a.b = "123"

	ac.KeepAlive(a.ac) // !
	runtime.GC()

	e := []byte(fmt.Sprintf("nihaocai %d", 1))
	c := NewSlice[*allocKeepTest1](a.ac, 0, 8)
	for i := 0; i < 1000000; i++ {
		b := New[allocKeepTest1](a.ac)
		b.b = a.ac.NewString(string(e)) // e 需要使用 NewString 来保留
		c = Append[*allocKeepTest1](a.ac, c, b)
	}
	runtime.GC()

	for i := 0; i < 1000000; i++ {
		assert.EqualValues(t, c[i].b, "nihaocai 1")
	}

	runtime.KeepAlive(ac)
}
