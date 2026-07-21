// Package decide decides the "correct" alias for each (package, path) pair
// observed by internal/scan: explicit config wins over majority vote, and an
// unresolved multi-way tie is left for a human to collapse.
//
// This experiment only covers FR-6.10 (one import path used with multiple
// aliases within a package). FR-6.11 (one alias used for multiple import
// paths) and FR-6.16 (duplicate same-path imports within a single file) are
// out of scope here; see docs/02notice.md round 6.
package decide

import (
	"sort"

	"github.com/podhmo/go-importalias/internal/shape"
)

// Options controls decision behavior.
type Options struct {
	// Strict, when true, treats any multiplicity of aliases for a path as a
	// tie regardless of vote counts.
	Strict bool
}

type pathKey struct{ pkg, path string }

// Decide groups occs by (package, path) and decides the "correct" alias for
// each group: config priority, then majority vote, then tie.
func Decide(occs []shape.Occurrence, cfg *shape.File, opts Options) []shape.Decision {
	byPath := map[pathKey][]shape.Occurrence{}
	for _, o := range occs {
		k := pathKey{o.Package, o.Path}
		byPath[k] = append(byPath[k], o)
	}

	keys := make([]pathKey, 0, len(byPath))
	for k := range byPath {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].pkg != keys[j].pkg {
			return keys[i].pkg < keys[j].pkg
		}
		return keys[i].path < keys[j].path
	})

	decisions := make([]shape.Decision, 0, len(keys))
	for _, k := range keys {
		decisions = append(decisions, decideOne(k.pkg, k.path, byPath[k], cfg, opts))
	}
	return decisions
}

func decideOne(pkg, path string, group []shape.Occurrence, cfg *shape.File, opts Options) shape.Decision {
	d := shape.Decision{Package: pkg, Path: path}

	if cfg != nil {
		if v, ok := cfg.Lookup(pkg, path); ok {
			if v.IsTie() {
				d.Tie = true
				d.TieCandidate = append([]string(nil), v.Tie...)
				sort.Strings(d.TieCandidate)
				return d
			}
			d.WantAlias = v.Resolved
			for _, o := range group {
				if o.Alias != v.Resolved {
					d.Inconsistent = append(d.Inconsistent, o)
				}
			}
			return d
		}
	}

	counts := map[string]int{}
	for _, o := range group {
		counts[o.Alias]++
	}
	distinct := make([]string, 0, len(counts))
	for a := range counts {
		distinct = append(distinct, a)
	}
	sort.Strings(distinct)

	if opts.Strict && len(distinct) > 1 {
		d.Tie = true
		d.TieCandidate = distinct
		return d
	}

	maxCount := 0
	for _, a := range distinct {
		if counts[a] > maxCount {
			maxCount = counts[a]
		}
	}
	var winners []string
	for _, a := range distinct {
		if counts[a] == maxCount {
			winners = append(winners, a)
		}
	}

	if len(winners) > 1 {
		d.Tie = true
		d.TieCandidate = winners
		return d
	}

	d.WantAlias = winners[0]
	for _, o := range group {
		if o.Alias != d.WantAlias {
			d.Inconsistent = append(d.Inconsistent, o)
		}
	}
	return d
}
