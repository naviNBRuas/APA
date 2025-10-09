package main

// main is required for the `wasi` target, even if it does nothing.
func main() {}

// add is a simple function that we will export from our WASM module.
//export add
func add(a, b uint32) uint32 {
	return a + b
}
