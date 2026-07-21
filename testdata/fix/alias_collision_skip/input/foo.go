package aliascollisionskip

import x "fmt"

func foo() {
	fmt := struct {
		Println func(...any) (int, error)
	}{}
	_ = fmt
	x.Println("foo")
}
