# typestate

Compile-time-safe builders for Go. Turn *"you forgot to set `Name`"* runtime
panics into **compile errors**, using phantom type parameters to track which
required fields a builder has been given.

Zero dependencies (the runtime is two marker types). Go 1.24+.

```go
import ts "github.com/zkrebbekx/typestate"
```

## The idea

Rust's typestate pattern enforces that a builder's required fields are set
before you can `build()`. Go's generics (1.18+) can express the same thing.
A builder carries **one type parameter per required field**, starting at
`Unset`. Each `WithX` flips its own slot to `Set`. The terminal constructor is
a free function pinned to the all-`Set` instantiation — so it simply does not
compile until every required field is present.

```go
build(New().WithURL("..."))                 // compile error: missing Name
build(New().WithURL("...").WithName("svc")) // ok
```

No runtime checks, no reflection, no `error` return for "field missing" — the
type checker is the guard.

## Worked example

```go
type Client struct {
	Name string // required
	URL  string // required
	Port int    // optional
}

// One Flag slot per required field. Optional fields get none.
type clientBuilder[Name, URL ts.Flag] struct{ c Client }

func newClient() clientBuilder[ts.Unset, ts.Unset] {
	return clientBuilder[ts.Unset, ts.Unset]{c: Client{Port: 8080}}
}

// Each method flips ONE slot to ts.Set; the others pass through.
// Flags are phantom (not stored), so the transition is a plain conversion.
func (b clientBuilder[N, U]) WithName(s string) clientBuilder[ts.Set, U] {
	b.c.Name = s
	return clientBuilder[ts.Set, U](b)
}
func (b clientBuilder[N, U]) WithURL(s string) clientBuilder[N, ts.Set] {
	b.c.URL = s
	return clientBuilder[N, ts.Set](b)
}
func (b clientBuilder[N, U]) WithPort(p int) clientBuilder[N, U] { // optional: no slot
	b.c.Port = p
	return b
}

// Terminal guard: only callable when BOTH required slots are ts.Set.
func build(b clientBuilder[ts.Set, ts.Set]) Client { return b.c }
```

Usage is fluent and order-independent:

```go
c := build(newClient().WithName("billing").WithURL("https://billing.svc"))
```

## Why the terminal guard is a free function

A Go type's method set is identical across every instantiation — you cannot
define `Build()` "only for `clientBuilder[Set, Set]`". So the all-`Set`
requirement lives on a **function parameter** instead:

```go
func build(b clientBuilder[ts.Set, ts.Set]) Client
```

Everything else (the `WithX` transitions) stays as fluent methods, because a
method's return type *may* re-instantiate the receiver's type parameters with
concrete markers.

## Design notes

- **Linear, not combinatorial.** Each `WithX` touches only its own slot, so a
  builder with N required fields needs N type params and N methods — no
  explosion.
- **Sealed flags.** `Flag` is an interface with an unexported method; only
  `Set` and `Unset` satisfy it, so callers cannot invent a third state.
- **Phantom transitions are free.** Because the flags never appear in struct
  fields, flipping a slot is a zero-cost type conversion.

## Verifying the guarantee

The happy paths are covered by `go test`. The *negative* cases (too few
required fields) cannot live in a normal test — they would not compile. They
live in `guard_compilefail.go` behind a build tag, and CI asserts that file
**fails** to build:

```sh
make test          # happy paths, race detector
make verify-guard  # asserts missing-field cases are rejected at compile time
make lint
```

## Code generation

Hand-writing the builder is fine for two or three required fields. Past that,
let the generator emit it. Install the tool and annotate required fields with
the `typestate:"required"` struct tag:

```go
//go:generate typestate -type Server

type Server struct {
	Addr    string        `typestate:"required"`
	Handler string        `typestate:"required"`
	Timeout time.Duration // optional — no tag, no type-param slot
	MaxConn int           // optional
}
```

```sh
go install github.com/zkrebbekx/typestate/cmd/typestate@latest
go generate ./...   # writes server_builder.go
```

It emits a `ServerBuilder[Addr, Handler ts.Flag]`, a `NewServerBuilder`, one
`WithX` per field, and the `BuildServer(b ServerBuilder[ts.Set, ts.Set])`
guard — exactly the shape shown above. Imports referenced by field types (like
`time`) are carried into the generated file. See [`examples/`](./examples) for
a runnable end-to-end setup (`make gen && make test`).

Rules the generator enforces:

- Only fields tagged `typestate:"required"` get a type-param slot.
- A required field must be exported (it has to be settable).
- Embedded fields are rejected (no simple name to address).
- A struct with zero required fields yields a plain, non-generic builder.

## Roadmap

- **v0.1:** the `Flag` / `Set` / `Unset` vocabulary plus a canonical, tested
  reference builder. Hand-written builders, ideal up to a handful of fields.
- **v0.2 (this release):** the `go:generate` tool above, emitting builders from
  annotated structs.

## License

MIT
