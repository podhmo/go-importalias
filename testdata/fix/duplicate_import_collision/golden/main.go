package duplicateimportcollision

import (
	f "fmt"
	oldfmt "fmt"
)

func usesCanonical() {
	f.Println("canonical")
}

func blocked() {
	f := struct{ Println func(string) }{Println: func(string) {}}
	oldfmt.Println("duplicate")
	_ = f
}
