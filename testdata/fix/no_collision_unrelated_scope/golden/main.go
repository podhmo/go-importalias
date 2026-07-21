package main

import (
	f "fmt"
)

func unrelated() {
	f := 1
	_ = f
}

func main() {
	f.Println("hello")
}
