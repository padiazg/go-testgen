package sample

import "context"

// SomeType is a local type used to test channel of named type.
type SomeType struct {
	ID   string
	Name string
}

// ChannelRecvReturn returns a receive-only channel of a named pointer type.
func ChannelRecvReturn(ctx context.Context) <-chan *SomeType {
	ch := make(chan *SomeType)
	go func() {
		defer close(ch)
		ch <- &SomeType{ID: "1", Name: "one"}
	}()
	return ch
}

// ChannelSendParam accepts a send-only channel parameter.
func ChannelSendParam(ch chan<- string) {
	ch <- "done"
}

// ChannelBidi accepts and returns a bidirectional channel.
func ChannelBidi(ch chan int) chan int {
	return ch
}

// ChannelSlice returns a receive-only channel of slices.
func ChannelSlice() <-chan []string {
	ch := make(chan []string)
	go func() {
		defer close(ch)
		ch <- []string{"a", "b", "c"}
	}()
	return ch
}

// ChannelPointer returns a bidirectional channel of pointers.
func ChannelPointer() chan *SomeType {
	return make(chan *SomeType)
}

// MultiChannelParam accepts multiple channel parameters.
func MultiChannelParam(
	input chan<- int,
	output chan int,
	recv <-chan string,
) {
	input <- 42
	output <- 99
	_ = <-recv
}

// NoChannelPlain is a plain function with no channel types (control case).
func NoChannelPlain(x int, y string) (int, error) {
	return x, nil
}
