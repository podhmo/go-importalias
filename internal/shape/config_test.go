package shape

import (
	"encoding/json"
	"testing"
)

func TestAliasValueRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		v    AliasValue
		json string
	}{
		{"resolved-empty", AliasValue{Resolved: ""}, `""`},
		{"resolved-name", AliasValue{Resolved: "errors"}, `"errors"`},
		{"tie-2", AliasValue{Tie: []string{"b", "a"}}, `["a","b"]`},
		{"tie-3", AliasValue{Tie: []string{"c", "a", "b"}}, `["a","b","c"]`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(c.v)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if got := string(data); got != c.json {
				t.Fatalf("Marshal = %s, want %s", got, c.json)
			}

			var got AliasValue
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got.Resolved != c.v.Resolved {
				t.Fatalf("Resolved = %q, want %q", got.Resolved, c.v.Resolved)
			}
			if len(got.Tie) != len(c.v.Tie) {
				t.Fatalf("Tie = %v, want same length as %v", got.Tie, c.v.Tie)
			}
		})
	}
}

func TestAliasValueUnmarshalInvalid(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"empty-array", `[]`},
		{"single-element-array", `["a"]`},
		{"number", `1`},
		{"object", `{}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var v AliasValue
			if err := json.Unmarshal([]byte(c.json), &v); err == nil {
				t.Fatalf("Unmarshal(%s) = nil error, want error", c.json)
			}
		})
	}
}

func TestNewFileMarshalsEmptyPackages(t *testing.T) {
	f := NewFile()
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(data), `{"packages":{}}`; got != want {
		t.Fatalf("Marshal(NewFile()) = %s, want %s", got, want)
	}

	var zero File
	data, err = json.Marshal(zero)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(data), `{"packages":null}`; got != want {
		t.Fatalf("Marshal(File{}) = %s, want %s (nil-map trap)", got, want)
	}
}

func TestFileLookupPriority(t *testing.T) {
	f := &File{
		Packages: map[string]map[string]AliasValue{
			"*": {
				"github.com/pkg/errors": {Resolved: "errors"},
			},
			"foo/...": {
				"github.com/aws/aws-sdk-go/aws": {Resolved: "awssdk"},
			},
			"foo/bar/baz": {
				"github.com/aws/aws-sdk-go/aws": {Resolved: "aws"},
			},
		},
	}

	cases := []struct {
		name   string
		pkg    string
		path   string
		want   AliasValue
		wantOk bool
	}{
		{"exact-pkg-wins", "foo/bar/baz", "github.com/aws/aws-sdk-go/aws", AliasValue{Resolved: "aws"}, true},
		{"prefix-scope-matches-subpackage", "foo/bar/qux", "github.com/aws/aws-sdk-go/aws", AliasValue{Resolved: "awssdk"}, true},
		{"prefix-scope-matches-self", "foo", "github.com/aws/aws-sdk-go/aws", AliasValue{Resolved: "awssdk"}, true},
		{"global-fallback", "other/pkg", "github.com/pkg/errors", AliasValue{Resolved: "errors"}, true},
		{"no-match", "other/pkg", "github.com/aws/aws-sdk-go/aws", AliasValue{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := f.Lookup(c.pkg, c.path)
			if ok != c.wantOk {
				t.Fatalf("Lookup(%q, %q) ok = %v, want %v", c.pkg, c.path, ok, c.wantOk)
			}
			if ok && got.Resolved != c.want.Resolved {
				t.Fatalf("Lookup(%q, %q) = %+v, want %+v", c.pkg, c.path, got, c.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	existing := &File{
		Ignore: []string{"foo/legacy"},
		Packages: map[string]map[string]AliasValue{
			"foo": {"a/path": {Resolved: "x"}, "b/path": {Resolved: "y"}},
			"bar": {"c/path": {Resolved: "z"}},
		},
	}
	fresh := &File{
		Packages: map[string]map[string]AliasValue{
			"foo": {"a/path": {Resolved: "x2"}}, // b/path dropped: whole-package replace
		},
	}

	merged := Merge(existing, fresh)

	if got, want := merged.Packages["foo"]["a/path"].Resolved, "x2"; got != want {
		t.Fatalf("foo/a/path = %q, want %q", got, want)
	}
	if _, ok := merged.Packages["foo"]["b/path"]; ok {
		t.Fatalf("foo/b/path should have been dropped by whole-package replace")
	}
	if got, want := merged.Packages["bar"]["c/path"].Resolved, "z"; got != want {
		t.Fatalf("bar/c/path (untouched by scan) = %q, want %q", got, want)
	}
	if len(merged.Ignore) != 1 || merged.Ignore[0] != "foo/legacy" {
		t.Fatalf("Ignore = %v, want carried over from existing", merged.Ignore)
	}
}
