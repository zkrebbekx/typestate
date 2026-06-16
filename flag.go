// Package typestate provides the type-level vocabulary for building
// compile-time-safe builders in Go: phantom "typestate" flags that track,
// at the type level, whether each required field of a builder has been set.
//
// The pattern lets you turn "you forgot to set X" runtime errors into
// compile errors. A builder carries one [Flag] type parameter per required
// field, starting at [Unset]. Each WithX method returns the builder with
// that slot re-instantiated to [Set]. The terminal constructor is a free
// function pinned to the all-[Set] instantiation, so it only compiles once
// every required field has been provided.
//
//	type clientBuilder[Name, URL typestate.Flag] struct{ c Client }
//
//	func (b clientBuilder[N, U]) WithName(s string) clientBuilder[typestate.Set, U]
//	func (b clientBuilder[N, U]) WithURL(s string)  clientBuilder[N, typestate.Set]
//
//	func build(b clientBuilder[typestate.Set, typestate.Set]) Client // guard
//
// See the package examples for a full, working builder.
//
// This package is the runtime vocabulary (v0.1). A go:generate code
// generator that emits these builders from an annotated struct is planned
// (v0.2); the generated code targets exactly the shape shown above.
package typestate

// Flag is the constraint for a typestate slot. It is a sealed interface:
// the only types that satisfy it are [Set] and [Unset], both defined in
// this package. This prevents callers from inventing a third state and
// keeps the "set or not set" guarantee total.
type Flag interface {
	// isFlag is unexported, so Flag cannot be satisfied outside this package.
	isFlag()
}

// Set marks a required builder slot as provided. A terminal constructor
// pinned to Set in every slot only compiles when every required field has
// been set.
type Set struct{}

// Unset marks a required builder slot as not yet provided. New builders
// start with every required slot at Unset.
type Unset struct{}

func (Set) isFlag()   {}
func (Unset) isFlag() {}
