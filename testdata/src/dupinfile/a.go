package dupinfile

import (
	f "fmt" // want `import "fmt" is imported multiple times in this file with different aliases`
	g "fmt" // want `import "fmt" is imported multiple times in this file with different aliases`
)

func A() {
	f.Println("a")
	g.Println("b")
}
