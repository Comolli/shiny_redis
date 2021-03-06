package collection

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

//wrapper sync.Once
//let sync.Once.do could recevicer a return param

type OncePointer struct {
	done uint32
	v    unsafe.Pointer
	m    sync.Mutex
}

func (o *OncePointer) Do(f func() unsafe.Pointer) unsafe.Pointer {
	if atomic.LoadUint32(&o.done) == 0 {
		o.m.Lock()
		defer o.m.Unlock()
		if o.done == 0 {
			defer atomic.StoreUint32(&o.done, 1)
			o.v = f()
		}
	}
	return o.v
}
