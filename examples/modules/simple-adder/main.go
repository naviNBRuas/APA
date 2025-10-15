package main

import "unsafe"

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export add
func add(a, b uint32) uint32 {
	return a + b
}

//export _start
func _start() {
	msg := "Hello from the simple-adder module!"
	ptr, size := stringToPtr(msg)
	log_message(ptr, size)
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
