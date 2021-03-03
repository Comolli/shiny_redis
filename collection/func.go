package collection

import (
	"runtime"
	"sync"
)

var fnPool sync.Pool

type fn struct {
	ctx  interface{}
	done chan struct{}
}

func getfn() *fn {
	v := fnPool.Get()
	if v == nil {
		v = &fn{
			done: make(chan struct{}, 1),
		}
	}
	return v.(*fn)
}

func fner(fnCh <-chan *fn, f func(ctx interface{})) {
	for fn := range fnCh {
		f(fn.ctx)
		fn.done <- struct{}{}
	}
}

func putfn(f *fn) {
	f.ctx = nil
	fnPool.Put(f)
}

func NewFunc(f func(ctx interface{})) func(ctx interface{}) bool {
	if f == nil {
		panic("BUG: f cannot be nil")
	}

	fnCh := make(chan *fn, runtime.GOMAXPROCS(-1)*2048)
	onceInit := func() {
		n := runtime.GOMAXPROCS(-1)
		for i := 0; i < n; i++ {
			go fner(fnCh, f)
		}
	}
	var once sync.Once

	return func(ctx interface{}) bool {
		once.Do(onceInit)
		fn := getfn()
		fn.ctx = ctx

		select {
		case fnCh <- fn:
		default:
			putfn(fn)
			return false
		}
		<-fn.done
		putfn(fn)
		return true
	}
}
