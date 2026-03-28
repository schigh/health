package syncmap_test

import (
	"sync"
	"testing"

	"github.com/schigh/health/internal/syncmap"
)

func TestMap_SetGet(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.Set("b", 2)

	v, ok := m.Get("a")
	if !ok || v != 1 {
		t.Fatalf("expected (1, true), got (%d, %t)", v, ok)
	}

	v, ok = m.Get("b")
	if !ok || v != 2 {
		t.Fatalf("expected (2, true), got (%d, %t)", v, ok)
	}

	_, ok = m.Get("c")
	if ok {
		t.Fatal("expected (_, false) for missing key")
	}
}

func TestMap_Delete(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.Delete("a")

	_, ok := m.Get("a")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestMap_Size(t *testing.T) {
	var m syncmap.Map[string, int]
	if m.Size() != 0 {
		t.Fatalf("expected size 0, got %d", m.Size())
	}
	m.Set("a", 1)
	m.Set("b", 2)
	if m.Size() != 2 {
		t.Fatalf("expected size 2, got %d", m.Size())
	}
}

func TestMap_Keys(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("x", 10)
	m.Set("y", 20)

	keys := m.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	found := map[string]bool{}
	for _, k := range keys {
		found[k] = true
	}
	if !found["x"] || !found["y"] {
		t.Fatalf("expected keys x and y, got %v", keys)
	}
}

func TestMap_Each(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	sum := 0
	m.Each(func(_ string, v int) bool {
		sum += v
		return true
	})
	if sum != 6 {
		t.Fatalf("expected sum 6, got %d", sum)
	}

	// test early termination
	count := 0
	m.Each(func(_ string, _ int) bool {
		count++
		return false
	})
	if count != 1 {
		t.Fatalf("expected 1 iteration with early stop, got %d", count)
	}
}

func TestMap_Clear(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.Clear()
	if m.Size() != 0 {
		t.Fatalf("expected empty map after clear, got size %d", m.Size())
	}
}

func TestMap_Value(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.Set("b", 2)

	v := m.Value()
	if len(v) != 2 || v["a"] != 1 || v["b"] != 2 {
		t.Fatalf("unexpected value map: %v", v)
	}

	// mutating the copy should not affect the original
	v["a"] = 99
	got, _ := m.Get("a")
	if got != 1 {
		t.Fatal("mutating Value() copy affected the original map")
	}
}

func TestMap_AbsorbMap(t *testing.T) {
	var m syncmap.Map[string, int]
	m.Set("a", 1)
	m.AbsorbMap(map[string]int{"b": 2, "c": 3})

	if m.Size() != 3 {
		t.Fatalf("expected size 3, got %d", m.Size())
	}
}

func TestMap_ConcurrentAccess(t *testing.T) {
	var m syncmap.Map[int, int]
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.Set(n, n*2)
			m.Get(n)
			m.Size()
			m.Keys()
		}(i)
	}
	wg.Wait()

	if m.Size() != 100 {
		t.Fatalf("expected 100 entries, got %d", m.Size())
	}
}
