//go:build wasip1

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"unsafe"
)

//go:wasmimport apa_host log_message
func log_message(ptr, size uint32)

//export hash_data
func hash_data() {
	dataToHash := "secret_password_123"
	
	hasher := sha256.New()
	hasher.Write([]byte(dataToHash))
	hash := hex.EncodeToString(hasher.Sum(nil))

	msg := "Hashed data from crypto-hasher module: " + hash
	logPtr, logSize := stringToPtr(msg)
	log_message(logPtr, logSize)
}

//export _start
func _start() {
	hash_data()
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	return uint32(uintptr(unsafe.Pointer(ptr))), uint32(len(buf))
}

func main() {}
