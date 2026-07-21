package main

import (
	"fmt"
)

func unrelated() {
	f := 1
	_ = f
}

func main() {
	fmt.Println("hello")
}
