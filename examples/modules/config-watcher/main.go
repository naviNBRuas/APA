//go:build wasip1

package main

import (
	"unsafe"
)

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export watch_config
func watch_config() {
	msg := "Config watcher module: Configuration file /etc/agent/config.yaml changed!"
	ptr, size := stringToPtr(msg)
	log_message(ptr, size)
}

//export _start
func _start() {
	watch_config()
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
