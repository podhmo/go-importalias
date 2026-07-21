package main

import f "fmt"

func main() {
	f.Println("before declaration")
	f := 1
	_ = f
}
