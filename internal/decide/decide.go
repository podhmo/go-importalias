// Package decide decides the "correct" alias for each (package, path) pair
// observed by internal/scan: explicit config wins over majority vote, and an
// unresolved multi-way tie is left for a human to collapse. It also detects
// FR-6.11 (one alias used for multiple import paths) as a separate axis.
//
// FR-6.16 (duplicate same-path imports within a single file) is out of scope
// here; see docs/02notice.md round 6.
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
type aliasKey struct{ pkg, alias string }

// Decide groups occs by (package, path) and decides the "correct" alias for
// each group (FR-6.10): config priority, then majority vote, then tie. It
// separately groups occs by (package, alias) to detect one alias mapped to
// more than one distinct import path (FR-6.11, DEC-11.12); occurrences with
// no alias are excluded from that axis.
func Decide(occs []shape.Occurrence, cfg *shape.File, opts Options) ([]shape.Decision, []shape.AliasCollision) {
	byPath := map[pathKey][]shape.Occurrence{}
	byAlias := map[aliasKey]map[string]shape.Occurrence{}
	for _, o := range occs {
		pk := pathKey{o.Package, o.Path}
		byPath[pk] = append(byPath[pk], o)

		if o.Alias == "" {
			continue
		}
		ak := aliasKey{o.Package, o.Alias}
		if byAlias[ak] == nil {
			byAlias[ak] = map[string]shape.Occurrence{}
		}
		if _, ok := byAlias[ak][o.Path]; !ok {
			byAlias[ak][o.Path] = o // DEC-11.16: first occurrence per path is the representative
		}
	}

	pathKeys := make([]pathKey, 0, len(byPath))
	for k := range byPath {
		pathKeys = append(pathKeys, k)
	}
	sort.Slice(pathKeys, func(i, j int) bool {
		if pathKeys[i].pkg != pathKeys[j].pkg {
			return pathKeys[i].pkg < pathKeys[j].pkg
		}
		return pathKeys[i].path < pathKeys[j].path
	})

	decisions := make([]shape.Decision, 0, len(pathKeys))
	for _, k := range pathKeys {
		decisions = append(decisions, decideOne(k.pkg, k.path, byPath[k], cfg, opts))
	}

	aliasKeys := make([]aliasKey, 0, len(byAlias))
	for k := range byAlias {
		aliasKeys = append(aliasKeys, k)
	}
	sort.Slice(aliasKeys, func(i, j int) bool {
		if aliasKeys[i].pkg != aliasKeys[j].pkg {
			return aliasKeys[i].pkg < aliasKeys[j].pkg
		}
		return aliasKeys[i].alias < aliasKeys[j].alias
	})

	var collisions []shape.AliasCollision
	for _, ak := range aliasKeys {
		byPathOcc := byAlias[ak]
		if len(byPathOcc) <= 1 {
			continue // same alias maps to exactly one path: not a collision
		}
		paths := make([]string, 0, len(byPathOcc))
		for p := range byPathOcc {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		occurrences := make([]shape.Occurrence, 0, len(paths))
		for _, p := range paths {
			occurrences = append(occurrences, byPathOcc[p])
		}
		collisions = append(collisions, shape.AliasCollision{
			Package:     ak.pkg,
			Alias:       ak.alias,
			Occurrences: occurrences,
		})
	}

	return decisions, collisions
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
