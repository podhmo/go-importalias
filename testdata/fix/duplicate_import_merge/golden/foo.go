package duplicateimportmerge

import (
	f "fmt"
)

func FromFoo() {
	f.Println("canonical")
	f.Println("duplicate")
}
