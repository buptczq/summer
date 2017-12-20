// Package startstop provides automatic Start/Stop for inject eliminating the
// necessity for manual ordering.
package summer

import (
	"bytes"
	"fmt"
	"sort"
)

// Opener defines the Open method, objects satisfying this interface will be
// opened by Start.
type Opener interface {
	Open() error
}

// Closer defines the Close method, objects satisfying this interface will be
// closed by Stop.
type Closer interface {
	Close() error
}

// Starter defines the Start method, objects satisfying this interface will be
// started by Start.
type Starter interface {
	Start() error
}

// Stopper defines the Stop method, objects satisfying this interface will be
// stopped by Stop.
type Stopper interface {
	Stop() error
}

// TryStart will start the graph, in the right order. It will call
// Start or Open. It returns the list of objects that have been
// successfully started. This can be used to stop only the
// dependencies that have been correctly started.
func (g *Graph)tryStart() ([]*Dew, error) {
	levels, err := levels(g.Objects())
	if err != nil {
		return nil, err
	}

	var started []*Dew
	for i := len(levels) - 1; i >= 0; i-- {
		level := levels[i]
		for _, o := range level {
			if openerO, ok := o.Value.(Opener); ok {
				if g.Logger != nil {
					g.Logger.Debugf("opening %s", o)
				}
				if err := openerO.Open(); err != nil {
					return started, err
				}
			}
			if starterO, ok := o.Value.(Starter); ok {
				if g.Logger != nil {
					g.Logger.Debugf("starting %s", o)
				}
				if err := starterO.Start(); err != nil {
					return started, err
				}
			}
			started = append(started, o)
		}
	}
	return started, nil
}

// Start the graph, in the right order. Start will call Start or Open if an
// object satisfies the associated interface.
func (g *Graph)Start() error {
	started, err := g.tryStart()
	g.started = started
	return err
}

// Stop the graph, in the right order. Stop will call Stop or Close if an
// object satisfies the associated interface.
func (g *Graph)Stop() error {
	levels, err := levels(g.started)
	if err != nil {
		return err
	}

	for _, level := range levels {
		for _, o := range level {
			if stopperO, ok := o.Value.(Stopper); ok {
				if g.Logger != nil {
					g.Logger.Debugf("stopping %s", o)
				}
				if err := stopperO.Stop(); err != nil {
					if g.Logger != nil {
						g.Logger.Errorf("error stopping %s: %s", o, err)
					}
					return err
				}
			}
			if closerO, ok := o.Value.(Closer); ok {
				if g.Logger != nil {
					g.Logger.Debugf("closing %s", o)
				}
				if err := closerO.Close(); err != nil {
					if g.Logger != nil {
						g.Logger.Errorf("error closing %s: %s", o, err)
					}
					return err
				}
			}
		}
	}
	return nil
}

// levels returns a slice of levels of objects of the Object Graph that
// implement Start/Stop.
func levels(objects []*Dew) ([][]*Dew, error) {
	levelsMap := map[int][]*Dew{}

	// ensure no cycles exist for objects that need start/stop, and make a
	// flattened graph of all deps.
	for _, o := range objects {
		if !isEligible(o) {
			continue
		}

		deps := map[*Dew]bool{}
		paths := allPaths(o, o, deps)
		for _, p := range paths {
			// special case direct cycle to itself
			if len(p) == 1 {
				return nil, cycleError(p)
			}

			// cycle is only relevant if more than one value in the path
			// isEligible. if there's just one, there isn't really a cycle from the
			// start/stop perspective.
			count := 0
			for _, s := range p {
				if isEligible(s.Object) {
					count++
				}
			}

			if count > 1 {
				return nil, cycleError(p)
			}
		}

		startStopDeps := 0
		for dep := range deps {
			if isEligible(dep) {
				startStopDeps++
			}
		}
		levelsMap[startStopDeps] = append(levelsMap[startStopDeps], o)
	}

	var levelsMapKeys []int
	for k := range levelsMap {
		levelsMapKeys = append(levelsMapKeys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levelsMapKeys)))

	levels := make([][]*Dew, 0, len(levelsMapKeys))
	for _, k := range levelsMapKeys {
		levels = append(levels, levelsMap[k])
	}
	return levels, nil
}

type path []struct {
	Field  string
	Object *Dew
}

type cycleError path

func (c cycleError) Error() string {
	var buf bytes.Buffer
	fmt.Fprint(&buf, "circular reference detected from")
	num := len(c)
	for _, s := range c {
		if num != 1 {
			fmt.Fprint(&buf, "\n")
		} else {
			fmt.Fprint(&buf, " ")
		}
		fmt.Fprintf(&buf, "field %s in %s", s.Field, s.Object)
	}
	if num == 1 {
		fmt.Fprint(&buf, " to itself")
	} else {
		fmt.Fprintf(&buf, "\nfield %s in %s", c[0].Field, c[0].Object)
	}
	return buf.String()
}

func allPaths(from, to *Dew, seen map[*Dew]bool) []path {
	if from != to {
		if seen[from] {
			return nil
		}
		seen[from] = true
	}

	var paths []path
	for field, value := range from.Fields {
		immediate := path{{Field: field, Object: from}}
		if value == to {
			paths = append(paths, immediate)
		} else {
			for _, p := range allPaths(value, to, seen) {
				paths = append(paths, append(immediate, p...))
			}
		}
	}
	return paths
}

func isEligible(i *Dew) bool {
	if _, ok := i.Value.(Starter); ok {
		return true
	}
	if _, ok := i.Value.(Stopper); ok {
		return true
	}
	if _, ok := i.Value.(Opener); ok {
		return true
	}
	if _, ok := i.Value.(Closer); ok {
		return true
	}
	return false
}
