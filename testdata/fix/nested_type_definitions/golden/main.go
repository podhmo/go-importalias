package nestedtypedefinitions

import f "fmt"

type FormatterAlias = interface {
	Format(f.State, rune)
	Nested() interface {
		Value() f.Stringer
		Struct() struct {
			Primary f.Stringer
			Next    interface{ Print(f.Stringer) }
		}
	}
}

type StructAlias = struct {
	Name f.Stringer
	Meta interface {
		Format(f.State, rune)
		Nested() struct{ Value f.Stringer }
	}
}
