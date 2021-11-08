package lib

import (
	"context"
	"log"
	"sync/atomic"
)

type EventListener interface {
	Handle(event Event)
}
type EventName string
type EventData interface{}

type Dispatcher struct {
	listener map[EventName]EventListener
	queue    chan Event
	queuing  *int32

	quit   context.Context
	Cancel context.CancelFunc
}

type Event struct {
	Name EventName
	Data EventData
}

func NewDispatcher() *Dispatcher {
	quit, cancel := context.WithCancel(context.Background())
	d := &Dispatcher{
		listener: map[EventName]EventListener{},
		queue:    make(chan Event),
		queuing:  new(int32),

		quit:   quit,
		Cancel: cancel,
	}
	go d.dispatching()
	return d
}

func (d *Dispatcher) dispatching() {
	for {
		select {
		case event := <-d.queue:
			if listener, ok := d.listener[event.Name]; ok {
				listener.Handle(event)
			} else {
				log.Printf("unregistered event '%s'", event.Name)
			}
		case <-d.quit.Done():
			atomic.CompareAndSwapInt32(d.queuing, 0, 1)
			return
		}
	}
}

func (d *Dispatcher) Register(listen EventListener, eventNames ...EventName) {
	for _, name := range eventNames {
		d.listener[name] = listen
	}
}

func (d *Dispatcher) Dispatch(event Event) {
	if atomic.LoadInt32(d.queuing) > 0 {
		log.Printf("already shutdown %+v [%T]", event, event)
	} else {
		d.queue <- event
	}
}
