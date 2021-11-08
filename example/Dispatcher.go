package main

import (
	"log"
	"time"

	"github.com/fiurthorn/go/lib"
)

type L struct{}

func (l *L) Handle(event lib.Event) {
	switch v := event.Data.(type) {
	case UserCreateEvent:
		log.Printf("Event(%s) [%T]: %+v", event.Name, v, v)
	default:
		log.Printf("Undefined(%s) [%T]: %+v", event.Name, v, v)
	}
}

type UserCreateEvent struct {
	Name     string
	Email    string
	Password string
}

func main() {
	l := &L{}

	d := lib.NewDispatcher()

	d.Register(l, "test")

	d.Dispatch(lib.Event{Name: "test", Data: struct{}{}})
	d.Dispatch(lib.Event{Name: "test", Data: UserCreateEvent{Name: "Fullname", Email: "test@export.com"}})
	d.Dispatch(lib.Event{Name: "test", Data: struct{}{}})
	d.Dispatch(lib.Event{Name: "test", Data: UserCreateEvent{Name: "Fullname", Email: "test@export.com"}})

	d.Cancel()
	for {
		time.Sleep(time.Second)
	}
}
