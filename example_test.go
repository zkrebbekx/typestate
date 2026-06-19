package typestate_test

import (
	"sync"
	"testing"

	ts "github.com/zkrebbekx/typestate"

	. "github.com/smartystreets/goconvey/convey"
)

// Client is the product type the builder constructs.
type Client struct {
	Name string // required
	URL  string // required
	Port int    // optional, defaults to 8080
}

// clientBuilder carries one type-level flag per *required* field. Optional
// fields need no flag. New builders start fully Unset; each WithX flips its
// own slot to ts.Set in the return type, leaving the others untouched.
type clientBuilder[Name, URL ts.Flag] struct {
	c Client
}

// newClient returns a builder with every required slot Unset and optional
// defaults applied.
func newClient() clientBuilder[ts.Unset, ts.Unset] {
	return clientBuilder[ts.Unset, ts.Unset]{c: Client{Port: 8080}}
}

// WithName satisfies the Name requirement: the returned builder has its
// first slot pinned to ts.Set while the URL slot (U) passes through.
func (b clientBuilder[N, U]) WithName(name string) clientBuilder[ts.Set, U] {
	b.c.Name = name
	// The flags are phantom (absent from the struct fields), so re-stating
	// the typestate is a plain conversion, not a rebuild.
	return clientBuilder[ts.Set, U](b)
}

// WithURL satisfies the URL requirement; the Name slot (N) passes through.
func (b clientBuilder[N, U]) WithURL(url string) clientBuilder[N, ts.Set] {
	b.c.URL = url
	return clientBuilder[N, ts.Set](b)
}

// WithPort sets the optional port. It touches no flag slot, so it can be
// called at any point (or not at all) without changing the typestate.
func (b clientBuilder[N, U]) WithPort(port int) clientBuilder[N, U] {
	b.c.Port = port
	return b
}

// build is the terminal guard. Because it is a free function pinned to the
// all-ts.Set instantiation, it only compiles when both required fields have
// been set. A method could not express this: a type's method set is the
// same across every instantiation, so the constraint must live on a
// function parameter.
func build(b clientBuilder[ts.Set, ts.Set]) Client {
	return b.c
}

func TestClientBuilder(t *testing.T) {
	Convey("Given a new client builder with both required fields unset", t, func() {
		b := newClient()

		Convey("When both required fields are set", func() {
			c := build(b.WithName("svc").WithURL("http://x"))

			Convey("Then the client carries those values and the optional default", func() {
				So(c.Name, ShouldEqual, "svc")
				So(c.URL, ShouldEqual, "http://x")
				So(c.Port, ShouldEqual, 8080)
			})
		})

		Convey("When the required fields are set in reverse order", func() {
			c := build(b.WithURL("http://x").WithName("svc"))

			Convey("Then it still builds, proving slot tracking is order-independent", func() {
				So(c.Name, ShouldEqual, "svc")
				So(c.URL, ShouldEqual, "http://x")
			})
		})

		Convey("When the optional port is overridden", func() {
			c := build(b.WithName("svc").WithURL("http://x").WithPort(9090))

			Convey("Then the override wins over the default", func() {
				So(c.Port, ShouldEqual, 9090)
			})
		})
	})
}

func TestConcurrentIndependentBuilders(t *testing.T) {
	Convey("Given many goroutines each building from a fresh builder", t, func() {
		const n = 50
		var wg sync.WaitGroup
		results := make([]Client, n)

		Convey("When they build concurrently (race detector on)", func() {
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					results[i] = build(newClient().WithName("svc").WithURL("u"))
				}(i)
			}
			wg.Wait()

			Convey("Then every result is correct with no data race", func() {
				for _, c := range results {
					So(c.Name, ShouldEqual, "svc")
					So(c.Port, ShouldEqual, 8080)
				}
			})
		})
	})
}

// Example shows the fluent, compile-time-checked construction.
func Example() {
	c := build(newClient().
		WithName("billing").
		WithURL("https://billing.svc").
		WithPort(443))

	_ = c
	// Output:
}
