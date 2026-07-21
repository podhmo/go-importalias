package dup

import (
	"fmt" // want `should use alias "f", not no alias`
)

func B() {
	fmt.Println("b")
}
