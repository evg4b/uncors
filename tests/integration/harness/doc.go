//go:build integration

// Package harness provides reusable infrastructure for uncors end-to-end
// integration tests: a request-recording backend, an in-process proxy launcher,
// and a TLS-trusting client factory.
//
// Everything here is gated behind the `integration` build tag so it never
// compiles into unit-test builds and never binds a socket unless explicitly
// requested via `go test -tags integration`.
package harness
