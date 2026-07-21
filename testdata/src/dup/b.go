package dup

import (
	"fmt"
)

func B() {
	fmt.Println("b1") // want `should use alias "f", not no alias`
	fmt.Println("b2") // want `should use alias "f", not no alias`
}
