package collection

import (
	"sync"
	"sync/atomic"
)

//wrapper sync.once
//let sync.Once.do could recevicer a return param
type Once struct {
	done uint32
	v    interface{}
	m    sync.Mutex
}

func (o *Once) Do(f func() interface{}) interface{} {
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
