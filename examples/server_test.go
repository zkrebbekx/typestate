package main

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGeneratedServerBuilder(t *testing.T) {
	Convey("Given the generated ServerBuilder", t, func() {
		Convey("When both required fields and an optional are set", func() {
			s := BuildServer(NewServerBuilder().
				WithAddr(":9000").
				WithHandler("router").
				WithTimeout(3 * time.Second))

			Convey("Then the built Server carries every value", func() {
				So(s.Addr, ShouldEqual, ":9000")
				So(s.Handler, ShouldEqual, "router")
				So(s.Timeout, ShouldEqual, 3*time.Second)
				So(s.MaxConn, ShouldEqual, 0)
			})
		})

		Convey("When required fields are set in reverse order", func() {
			s := BuildServer(NewServerBuilder().
				WithHandler("router").
				WithAddr(":9000"))

			Convey("Then slot tracking is order-independent", func() {
				So(s.Addr, ShouldEqual, ":9000")
				So(s.Handler, ShouldEqual, "router")
			})
		})
	})
}
