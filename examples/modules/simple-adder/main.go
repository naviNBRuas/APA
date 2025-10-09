package main

//export add
func add(a, b uint32) uint32 {
	return a + b
}

//export _start
func _start() {
	// Empty _start function for now
}

func main() {}
