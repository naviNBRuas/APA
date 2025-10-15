//go:build wasip1

package main

import (
	"unsafe"
)

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export publish_message
func publish_message() {
	msg := "Published message from message-broker module: Hello from WASM!"
	logPtr, logSize := stringToPtr(msg)
	log_message(logPtr, logSize)
}

//export subscribe_to_topic
func subscribe_to_topic() {
	msg := "Subscribed to topic from message-broker module: my_topic"
	logPtr, logSize := stringToPtr(msg)
	log_message(logPtr, logSize)
}

//export _start
func _start() {
	publish_message()
	subscribe_to_topic()
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
