//go:build typestate_compilefail

// This file is NOT part of the normal build. It exists to prove the
// compile-time guarantee: building it must FAIL. The Makefile target
// `verify-guard` runs `go build -tags typestate_compilefail` and asserts a
// non-zero exit. If this file ever compiles, the typestate guard is broken.
//
// Each case below sets too few required fields and then calls the terminal
// guard, which is pinned to the all-Set instantiation.

package typestate

type guardClient struct {
	name string
	url  string
}

type guardBuilder[Name, URL Flag] struct{ c guardClient }

func newGuard() guardBuilder[Unset, Unset] { return guardBuilder[Unset, Unset]{} }

func (b guardBuilder[N, U]) withName(s string) guardBuilder[Set, U] {
	b.c.name = s
	return guardBuilder[Set, U]{c: b.c}
}

func (b guardBuilder[N, U]) withURL(s string) guardBuilder[N, Set] {
	b.c.url = s
	return guardBuilder[N, Set]{c: b.c}
}

func buildGuard(b guardBuilder[Set, Set]) guardClient { return b.c }

func init() {
	// Want: guardBuilder[Set, Set]. Got: guardBuilder[Unset, Unset].
	_ = buildGuard(newGuard())

	// Want: guardBuilder[Set, Set]. Got: guardBuilder[Set, Unset] (URL missing).
	_ = buildGuard(newGuard().withName("x"))

	// Want: guardBuilder[Set, Set]. Got: guardBuilder[Unset, Set] (Name missing).
	_ = buildGuard(newGuard().withURL("x"))
}
