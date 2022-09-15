// Package ringbuf provides a ring buffer data structure.
//
// # Ring buffer
//
// A ring buffer is a data structure that uses a single, fixed-size buffer
// as if it were connected end-to-end. This structure lends itself easily
// to buffering data streams.
//
// It is also known as circular buffer, circular queue, cyclic buffer.
//
// # Generics
//
// ringbuf uses generics to create a RingBuffer that contains items of the type
// specified. To create a RingBuffer that holds a specific type, provide a type
// argument to New or with the variable declaration.
package ringbuf
