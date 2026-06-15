//go:build integration

// Package integration provides end-to-end test infrastructure for uncors:
// a recording backend, an in-process proxy, and a client that trusts the proxy's
// dev CA. All teardown is registered via t.Cleanup — nothing to defer at call sites.
package integration
