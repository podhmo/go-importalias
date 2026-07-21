package duplicateimportmerge

import (
	f "fmt"
	oldfmt "fmt"
)

func FromFoo() {
	f.Println("canonical")
	oldfmt.Println("duplicate")
}
