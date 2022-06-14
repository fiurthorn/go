package lib

import (
	"runtime"
	"sync/atomic"
)

type TicketStore[Value any] struct {
	ticket uint64
	done   uint64
	slots  []Value
}

func NewTicketStore[Value any]() *TicketStore[Value] {
	return &TicketStore[Value]{
		ticket: 0,
		done:   0,
	}
}

func (ts *TicketStore[Value]) Put(s Value) {
	t := atomic.AddUint64(&ts.ticket, 1) - 1
	ts.slots[t] = s

	for !atomic.CompareAndSwapUint64(&ts.done, t, t+1) {
		runtime.Gosched()
	}
}

func (ts *TicketStore[Value]) GetDone() []Value {
	return ts.slots[:atomic.LoadUint64(&ts.done)+1]
}
