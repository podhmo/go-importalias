package samealias

import (
	x "fmt" // want `alias "x" is used for multiple import paths in this package`
)

func A() {
	x.Println("a")
}
