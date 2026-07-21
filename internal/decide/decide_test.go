package decide_test

import (
	"reflect"
	"testing"

	"github.com/podhmo/go-importalias/internal/decide"
	"github.com/podhmo/go-importalias/internal/shape"
)

func TestDecide_AliasCollisions(t *testing.T) {
	cases := []struct {
		name string
		occs []shape.Occurrence
		want []shape.AliasCollision
	}{
		{
			name: "single_path_no_collision",
			occs: []shape.Occurrence{
				{Package: "p", Path: "fmt", Alias: "x", Pos: 1},
			},
			want: nil,
		},
		{
			name: "alias_maps_to_two_paths",
			occs: []shape.Occurrence{
				{Package: "p", Path: "os", Alias: "x", Pos: 2},
				{Package: "p", Path: "fmt", Alias: "x", Pos: 1},
			},
			want: []shape.AliasCollision{
				{
					Package: "p",
					Alias:   "x",
					Occurrences: []shape.Occurrence{
						{Package: "p", Path: "fmt", Alias: "x", Pos: 1},
						{Package: "p", Path: "os", Alias: "x", Pos: 2},
					},
				},
			},
		},
		{
			name: "unaliased_excluded",
			occs: []shape.Occurrence{
				{Package: "p", Path: "fmt", Alias: "", Pos: 1},
				{Package: "p", Path: "os", Alias: "", Pos: 2},
			},
			want: nil,
		},
		{
			name: "collision_scoped_per_package",
			occs: []shape.Occurrence{
				{Package: "p1", Path: "fmt", Alias: "x", Pos: 1},
				{Package: "p2", Path: "os", Alias: "x", Pos: 2},
			},
			want: nil,
		},
		{
			name: "duplicate_occurrence_same_path_dedup",
			occs: []shape.Occurrence{
				{Package: "p", Path: "fmt", Alias: "x", Pos: 1},
				{Package: "p", Path: "fmt", Alias: "x", Pos: 99},
				{Package: "p", Path: "os", Alias: "x", Pos: 2},
			},
			want: []shape.AliasCollision{
				{
					Package: "p",
					Alias:   "x",
					Occurrences: []shape.Occurrence{
						{Package: "p", Path: "fmt", Alias: "x", Pos: 1},
						{Package: "p", Path: "os", Alias: "x", Pos: 2},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, got := decide.Decide(c.occs, nil, decide.Options{})
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("collisions = %+v, want %+v", got, c.want)
			}
		})
	}
}
