package cache

import (
	"context"
	"github.com/miekg/dns"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func Test_memCache(t *testing.T) {
	ctx := context.Background()

	c := newMemCache(8, 16, -1)
	for i := 0; i < 1024; i++ {
		key := strconv.Itoa(i)
		m := new(dns.Msg)
		m.Id = uint16(i)
		if err := c.store(ctx, key, m, time.Millisecond*200); err != nil {
			t.Fatal(err)
		}

		v, _, ok, err := c.get(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal()
		}
		if v.Id != uint16(i) {
			t.Fatal("cache kv mismatched")
		}
	}

	if c.len() > 8*16 {
		t.Fatal("cache overflow")
	}
}

func Test_memCache_cleaner(t *testing.T) {
	c := newMemCache(2, 8, time.Millisecond*10)
	defer c.Close()
	ctx := context.Background()
	for i := 0; i < 64; i++ {
		key := strconv.Itoa(i)
		m := new(dns.Msg)
		m.Id = uint16(i)
		if err := c.store(ctx, key, m, time.Millisecond*10); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Millisecond * 100)
	if c.len() != 0 {
		t.Fatal()
	}
}

func Test_memCache_race(t *testing.T) {
	c := newMemCache(32, 128, -1)
	defer c.Close()
	ctx := context.Background()

	m := &dns.Msg{}

	wg := sync.WaitGroup{}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 256; i++ {
				err := c.store(ctx, strconv.Itoa(i), m, time.Minute)
				if err != nil {
					t.Log(err)
					t.Fail()
				}
				v, _, ok, err := c.get(ctx, strconv.Itoa(i))
				if err != nil {
					t.Log(err)
					t.Fail()
					runtime.Goexit()
				}
				if !ok {
					t.Log("failed to get stored value")
					t.Fail()
					runtime.Goexit()
				}
				v.Id = uint16(i)
				c.lru.Clean(cleanFunc)
			}
		}()
	}
	wg.Wait()
}
