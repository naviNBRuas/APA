//go:build wasip1

package main

import (
	"unsafe"
)

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export log_data
func log_data(ptr, size uint32) {
	log_message(ptr, size)
}

//export _start
func _start() {
	msg := "Data from data-logger module: Some important data here!"
	ptr, size := stringToPtr(msg)
	log_data(ptr, size)
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
