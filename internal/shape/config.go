package shape

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"
)

// File is the on-disk config file (importalias.json).
type File struct {
	Packages map[string]map[string]AliasValue `json:"packages"`
	Ignore   []string                         `json:"ignore,omitempty"`
}

// NewFile returns a File with initialized (non-nil) maps, so it marshals as
// "packages": {} instead of "packages": null.
func NewFile() *File {
	return &File{Packages: map[string]map[string]AliasValue{}}
}

// AliasValue is either a single resolved alias ("" means "no alias"), or an
// unresolved tie between 2 or more candidate aliases.
type AliasValue struct {
	Resolved string
	Tie      []string
}

// IsTie reports whether this value represents an unresolved tie.
func (v AliasValue) IsTie() bool {
	return len(v.Tie) > 0
}

// MarshalJSON encodes a resolved value as a JSON string, and a tie as a JSON
// array of 2 or more strings.
func (v AliasValue) MarshalJSON() ([]byte, error) {
	if v.IsTie() {
		tie := append([]string(nil), v.Tie...)
		sort.Strings(tie)
		return json.Marshal(tie)
	}
	return json.Marshal(v.Resolved)
}

// UnmarshalJSON decodes either a JSON string (resolved) or a JSON array of 2+
// strings (tie). Any other shape is an error.
func (v *AliasValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = AliasValue{Resolved: s}
		return nil
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("alias value must be a string or an array of 2 or more strings: %w", err)
	}
	if len(arr) < 2 {
		return fmt.Errorf("alias value array must have 2 or more elements, got %d", len(arr))
	}
	*v = AliasValue{Tie: arr}
	return nil
}

// Load reads and parses the config file at path. A missing file is not an
// error; it returns an empty File instead.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	f := NewFile()
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if f.Packages == nil {
		f.Packages = map[string]map[string]AliasValue{}
	}
	return f, nil
}

// Save writes f to path as indented JSON with a trailing newline, with
// packages/scopes/paths sorted for stable diffs (map key order in
// encoding/json is already alphabetical; only Tie needs explicit sorting,
// handled in MarshalJSON).
func Save(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// Lookup resolves the config value for (pkg, path), applying the
// scope-priority rule: exact package name > longest-matching "foo/..." scope
// > "*" global. The second return value is false if no scope matches at all.
func (f *File) Lookup(pkg, path string) (AliasValue, bool) {
	if f == nil {
		return AliasValue{}, false
	}

	if scope, ok := f.Packages[pkg]; ok {
		if v, ok := scope[path]; ok {
			return v, true
		}
	}

	bestPrefixLen := -1
	var bestScope map[string]AliasValue
	for key, scope := range f.Packages {
		prefix, ok := strings.CutSuffix(key, "/...")
		if !ok {
			continue
		}
		if pkg != prefix && !strings.HasPrefix(pkg, prefix+"/") {
			continue
		}
		if len(prefix) > bestPrefixLen {
			bestPrefixLen = len(prefix)
			bestScope = scope
		}
	}
	if bestScope != nil {
		if v, ok := bestScope[path]; ok {
			return v, true
		}
	}

	if scope, ok := f.Packages["*"]; ok {
		if v, ok := scope[path]; ok {
			return v, true
		}
	}

	return AliasValue{}, false
}

// IgnoresPackage reports whether pkg matches any ignore pattern. Patterns use
// the same package-scope syntax as Lookup: exact package, "foo/..." prefix, or
// "*" global.
func (f *File) IgnoresPackage(pkg string) bool {
	if f == nil {
		return false
	}
	for _, pattern := range f.Ignore {
		if matchesPackageScope(pattern, pkg) {
			return true
		}
	}
	return false
}

func matchesPackageScope(pattern, pkg string) bool {
	if pattern == "*" || pattern == pkg {
		return true
	}
	prefix, ok := strings.CutSuffix(pattern, "/...")
	if !ok {
		return false
	}
	return pkg == prefix || strings.HasPrefix(pkg, prefix+"/")
}

// Merge combines an existing config with freshly scanned results: package
// keys present in fresh replace the existing entry wholesale (including any
// path-level sub-entries not re-observed in this scan); package keys absent
// from fresh are carried over unchanged. Ignore is always carried over from
// existing.
func Merge(existing, fresh *File) *File {
	out := NewFile()
	if existing != nil {
		out.Ignore = existing.Ignore
		maps.Copy(out.Packages, existing.Packages)
	}
	if fresh != nil {
		maps.Copy(out.Packages, fresh.Packages)
	}
	return out
}
