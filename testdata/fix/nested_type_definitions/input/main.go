package nestedtypedefinitions

import "fmt"

type FormatterAlias = interface {
	Format(fmt.State, rune)
	Nested() interface {
		Value() fmt.Stringer
		Struct() struct {
			Primary fmt.Stringer
			Next    interface{ Print(fmt.Stringer) }
		}
	}
}

type StructAlias = struct {
	Name fmt.Stringer
	Meta interface {
		Format(fmt.State, rune)
		Nested() struct{ Value fmt.Stringer }
	}
}
