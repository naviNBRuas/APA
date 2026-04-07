//go:build wasip1

package main

import (
	"encoding/json"
	"runtime"
	"unsafe"
)

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export get_system_info
func get_system_info() {
	info := struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
		CPUs int    `json:"cpus"`
	}{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		CPUs: runtime.NumCPU(),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return
	}

	ptr, size := stringToPtr(string(data))
	log_message(ptr, size)
}

//export _start
func _start() {
	get_system_info()
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
