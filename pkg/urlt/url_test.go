// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlt

import (
	"bytes"
	encodingPkg "encoding"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net"
	base_url "net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type URL = base_url.URL

type URLTest struct {
	in        string
	out       *URL   // expected parse
	roundtrip string // expected result of reserializing the URL; empty means same as "in".
}

var urltests = []URLTest{
	// no path
	{
		"http://www.google.com",
		&URL{
			Scheme: "http",
			Host:   "www.google.com",
		},
		"",
	},
	// path
	{
		"http://www.google.com/",
		&URL{
			Scheme: "http",
			Host:   "www.google.com",
			Path:   "/",
		},
		"",
	},
	// path with hex escaping
	{
		"http://www.google.com/file%20one%26two",
		&URL{
			Scheme:  "http",
			Host:    "www.google.com",
			Path:    "/file one&two",
			RawPath: "/file%20one%26two",
		},
		"",
	},
	// fragment with hex escaping
	{
		"http://www.google.com/#file%20one%26two",
		&URL{
			Scheme:      "http",
			Host:        "www.google.com",
			Path:        "/",
			Fragment:    "file one&two",
			RawFragment: "file%20one%26two",
		},
		"",
	},
	// user
	{
		"ftp://webmaster@www.google.com/",
		&URL{
			Scheme: "ftp",
			User:   base_url.User("webmaster"),
			Host:   "www.google.com",
			Path:   "/",
		},
		"",
	},
	// escape sequence in username
	{
		"ftp://john%20doe@www.google.com/",
		&URL{
			Scheme: "ftp",
			User:   base_url.User("john doe"),
			Host:   "www.google.com",
			Path:   "/",
		},
		"ftp://john%20doe@www.google.com/",
	},
	// empty query
	{
		"http://www.google.com/?",
		&URL{
			Scheme:     "http",
			Host:       "www.google.com",
			Path:       "/",
			ForceQuery: true,
		},
		"",
	},
	// query ending in question mark (Issue 14573)
	{
		"http://www.google.com/?foo=bar?",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/",
			RawQuery: "foo=bar?",
		},
		"",
	},
	// query
	{
		"http://www.google.com/?q=go+language",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/",
			RawQuery: "q=go+language",
		},
		"",
	},
	// query with hex escaping: NOT parsed
	{
		"http://www.google.com/?q=go%20language",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/",
			RawQuery: "q=go%20language",
		},
		"",
	},
	// %20 outside query
	{
		"http://www.google.com/a%20b?q=c+d",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/a b",
			RawQuery: "q=c+d",
		},
		"",
	},
	// path without leading /, so no parsing
	{
		"http:www.google.com/?q=go+language",
		&URL{
			Scheme:   "http",
			Opaque:   "www.google.com/",
			RawQuery: "q=go+language",
		},
		"http:www.google.com/?q=go+language",
	},
	// path without leading /, so no parsing
	{
		"http:%2f%2fwww.google.com/?q=go+language",
		&URL{
			Scheme:   "http",
			Opaque:   "%2f%2fwww.google.com/",
			RawQuery: "q=go+language",
		},
		"http:%2f%2fwww.google.com/?q=go+language",
	},
	// non-authority with path; see golang.org/issue/46059
	{
		"mailto:/webmaster@golang.org",
		&URL{
			Scheme:   "mailto",
			Path:     "/webmaster@golang.org",
			OmitHost: true,
		},
		"",
	},
	// non-authority
	{
		"mailto:webmaster@golang.org",
		&URL{
			Scheme: "mailto",
			Opaque: "webmaster@golang.org",
		},
		"",
	},
	// unescaped :// in query should not create a scheme
	{
		"/foo?query=http://bad",
		&URL{
			Path:     "/foo",
			RawQuery: "query=http://bad",
		},
		"",
	},
	// leading // without scheme should create an authority
	{
		"//foo",
		&URL{
			Host: "foo",
		},
		"",
	},
	// leading // without scheme, with userinfo, path, and query
	{
		"//user@foo/path?a=b",
		&URL{
			User:     base_url.User("user"),
			Host:     "foo",
			Path:     "/path",
			RawQuery: "a=b",
		},
		"",
	},
	// Three leading slashes isn't an authority, but doesn't return an error.
	// (We can't return an error, as this code is also used via
	// ServeHTTP -> ReadRequest -> Parse, which is arguably a
	// different URL parsing context, but currently shares the
	// same codepath)
	{
		"///threeslashes",
		&URL{
			Path: "///threeslashes",
		},
		"",
	},
	{
		"http://user:password@google.com",
		&URL{
			Scheme: "http",
			User:   base_url.UserPassword("user", "password"),
			Host:   "google.com",
		},
		"http://user:password@google.com",
	},
	// unescaped @ in username should not confuse host
	{
		"http://j@ne:password@google.com",
		&URL{
			Scheme: "http",
			User:   base_url.UserPassword("j@ne", "password"),
			Host:   "google.com",
		},
		"http://j%40ne:password@google.com",
	},
	// unescaped @ in password should not confuse host
	{
		"http://jane:p@ssword@google.com",
		&URL{
			Scheme: "http",
			User:   base_url.UserPassword("jane", "p@ssword"),
			Host:   "google.com",
		},
		"http://jane:p%40ssword@google.com",
	},
	{
		"http://j@ne:password@google.com/p@th?q=@go",
		&URL{
			Scheme:   "http",
			User:     base_url.UserPassword("j@ne", "password"),
			Host:     "google.com",
			Path:     "/p@th",
			RawQuery: "q=@go",
		},
		"http://j%40ne:password@google.com/p@th?q=@go",
	},
	{
		"http://www.google.com/?q=go+language#foo",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/",
			RawQuery: "q=go+language",
			Fragment: "foo",
		},
		"",
	},
	{
		"http://www.google.com/?q=go+language#foo&bar",
		&URL{
			Scheme:   "http",
			Host:     "www.google.com",
			Path:     "/",
			RawQuery: "q=go+language",
			Fragment: "foo&bar",
		},
		"http://www.google.com/?q=go+language#foo&bar",
	},
	{
		"http://www.google.com/?q=go+language#foo%26bar",
		&URL{
			Scheme:      "http",
			Host:        "www.google.com",
			Path:        "/",
			RawQuery:    "q=go+language",
			Fragment:    "foo&bar",
			RawFragment: "foo%26bar",
		},
		"http://www.google.com/?q=go+language#foo%26bar",
	},
	{
		"file:///home/adg/rabbits",
		&URL{
			Scheme: "file",
			Host:   "",
			Path:   "/home/adg/rabbits",
		},
		"file:///home/adg/rabbits",
	},
	// "Windows" paths are no exception to the rule.
	// See golang.org/issue/6027, especially comment #9.
	{
		"file:///C:/FooBar/Baz.txt",
		&URL{
			Scheme: "file",
			Host:   "",
			Path:   "/C:/FooBar/Baz.txt",
		},
		"file:///C:/FooBar/Baz.txt",
	},
	// case-insensitive scheme
	{
		"MaIlTo:webmaster@golang.org",
		&URL{
			Scheme: "mailto",
			Opaque: "webmaster@golang.org",
		},
		"mailto:webmaster@golang.org",
	},
	// Relative path
	{
		"a/b/c",
		&URL{
			Path: "a/b/c",
		},
		"a/b/c",
	},
	// escaped '?' in username and password
	{
		"http://%3Fam:pa%3Fsword@google.com",
		&URL{
			Scheme: "http",
			User:   base_url.UserPassword("?am", "pa?sword"),
			Host:   "google.com",
		},
		"",
	},
	// host subcomponent; IPv4 address in RFC 3986
	{
		"http://192.168.0.1/",
		&URL{
			Scheme: "http",
			Host:   "192.168.0.1",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv4 address in RFC 3986
	{
		"http://192.168.0.1:8080/",
		&URL{
			Scheme: "http",
			Host:   "192.168.0.1:8080",
			Path:   "/",
		},
		"",
	},
	// host subcomponent; IPv6 address in RFC 3986
	{
		"http://[fe80::1]/",
		&URL{
			Scheme: "http",
			Host:   "[fe80::1]",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv6 address in RFC 3986
	{
		"http://[fe80::1]:8080/",
		&URL{
			Scheme: "http",
			Host:   "[fe80::1]:8080",
			Path:   "/",
		},
		"",
	},
	// valid IPv6 host with port and path
	{
		"https://[2001:db8::1]:8443/test/path",
		&URL{
			Scheme: "https",
			Host:   "[2001:db8::1]:8443",
			Path:   "/test/path",
		},
		"",
	},
	// host subcomponent; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25en0]/", // alphanum zone identifier
		&URL{
			Scheme: "http",
			Host:   "[fe80::1%en0]",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25en0]:8080/", // alphanum zone identifier
		&URL{
			Scheme: "http",
			Host:   "[fe80::1%en0]:8080",
			Path:   "/",
		},
		"",
	},
	// host subcomponent; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25%65%6e%301-._~]/", // percent-encoded+unreserved zone identifier
		&URL{
			Scheme: "http",
			Host:   "[fe80::1%en01-._~]",
			Path:   "/",
		},
		"http://[fe80::1%25en01-._~]/",
	},
	// host and port subcomponents; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25%65%6e%301-._~]:8080/", // percent-encoded+unreserved zone identifier
		&URL{
			Scheme: "http",
			Host:   "[fe80::1%en01-._~]:8080",
			Path:   "/",
		},
		"http://[fe80::1%25en01-._~]:8080/",
	},
	// alternate escapings of path survive round trip
	{
		"http://rest.rsc.io/foo%2fbar/baz%2Fquux?alt=media",
		&URL{
			Scheme:   "http",
			Host:     "rest.rsc.io",
			Path:     "/foo/bar/baz/quux",
			RawPath:  "/foo%2fbar/baz%2Fquux",
			RawQuery: "alt=media",
		},
		"",
	},
	// issue 12036
	{
		"mysql://a,b,c/bar",
		&URL{
			Scheme: "mysql",
			Host:   "a,b,c",
			Path:   "/bar",
		},
		"",
	},
	// worst case host, still round trips
	{
		"scheme://!$&'()*+,;=hello!:1/path",
		&URL{
			Scheme: "scheme",
			Host:   "!$&'()*+,;=hello!:1",
			Path:   "/path",
		},
		"",
	},
	// worst case path, still round trips
	{
		"http://host/!$&'()*+,;=:@[hello]",
		&URL{
			Scheme:  "http",
			Host:    "host",
			Path:    "/!$&'()*+,;=:@[hello]",
			RawPath: "/!$&'()*+,;=:@[hello]",
		},
		"",
	},
	// golang.org/issue/5684
	{
		"http://example.com/oid/[order_id]",
		&URL{
			Scheme:  "http",
			Host:    "example.com",
			Path:    "/oid/[order_id]",
			RawPath: "/oid/[order_id]",
		},
		"",
	},
	// golang.org/issue/12200 (colon with empty port)
	{
		"http://192.168.0.2:8080/foo",
		&URL{
			Scheme: "http",
			Host:   "192.168.0.2:8080",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://192.168.0.2:/foo",
		&URL{
			Scheme: "http",
			Host:   "192.168.0.2:",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:8080/foo",
		&URL{
			Scheme: "http",
			Host:   "[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:8080",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:/foo",
		&URL{
			Scheme: "http",
			Host:   "[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:",
			Path:   "/foo",
		},
		"",
	},
	// golang.org/issue/7991 and golang.org/issue/12719 (non-ascii %-encoded in host)
	{
		"http://hello.世界.com/foo",
		&URL{
			Scheme: "http",
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
	},
	{
		"http://hello.%e4%b8%96%e7%95%8c.com/foo",
		&URL{
			Scheme: "http",
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
	},
	{
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
		&URL{
			Scheme: "http",
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"",
	},
	// golang.org/issue/10433 (path beginning with //)
	{
		"http://example.com//foo",
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Path:   "//foo",
		},
		"",
	},
	// test that we can reparse the host names we accept.
	{
		"myscheme://authority<\"hi\">/foo",
		&URL{
			Scheme: "myscheme",
			Host:   "authority<\"hi\">",
			Path:   "/foo",
		},
		"",
	},
	// spaces in hosts are disallowed but escaped spaces in IPv6 scope IDs are grudgingly OK.
	// This happens on Windows.
	// golang.org/issue/14002
	{
		"tcp://[2020::2020:20:2020:2020%25Windows%20Loves%20Spaces]:2020",
		&URL{
			Scheme: "tcp",
			Host:   "[2020::2020:20:2020:2020%Windows Loves Spaces]:2020",
		},
		"",
	},
	// test we can roundtrip magnet url
	// fix issue https://golang.org/issue/20054
	{
		"magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn",
		&URL{
			Scheme:   "magnet",
			Host:     "",
			Path:     "",
			RawQuery: "xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn",
		},
		"magnet:?xt=urn:btih:c12fe1c06bba254a9dc9f519b335aa7c1367a88a&dn",
	},
	{
		"mailto:?subject=hi",
		&URL{
			Scheme:   "mailto",
			Host:     "",
			Path:     "",
			RawQuery: "subject=hi",
		},
		"mailto:?subject=hi",
	},
	// PostgreSQL URLs can include a comma-separated list of host:post hosts.
	// https://go.dev/issue/75859
	{
		"postgres://host1:1,host2:2,host3:3",
		&URL{
			Scheme: "postgres",
			Host:   "host1:1,host2:2,host3:3",
			Path:   "",
		},
		"postgres://host1:1,host2:2,host3:3",
	},
	{
		"postgresql://host1:1,host2:2,host3:3",
		&URL{
			Scheme: "postgresql",
			Host:   "host1:1,host2:2,host3:3",
			Path:   "",
		},
		"postgresql://host1:1,host2:2,host3:3",
	},
	// Mongodb URLs can include a comma-separated list of host:post hosts.
	{
		"mongodb://user:password@host1:1,host2:2,host3:3",
		&URL{
			Scheme: "mongodb",
			User:   base_url.UserPassword("user", "password"),
			Host:   "host1:1,host2:2,host3:3",
			Path:   "",
		},
		"",
	},
	{
		"mongodb+srv://user:password@host1:1,host2:2,host3:3",
		&URL{
			Scheme: "mongodb+srv",
			User:   base_url.UserPassword("user", "password"),
			Host:   "host1:1,host2:2,host3:3",
			Path:   "",
		},
		"",
	},
	// {client} placeholder as full host
	{
		"http://{client}/",
		&URL{
			Scheme: "http",
			Host:   "{client}",
			Path:   "/",
		},
		"",
	},
	// {client} placeholder as subdomain
	{
		"http://{client}.example.com/path",
		&URL{
			Scheme: "http",
			Host:   "{client}.example.com",
			Path:   "/path",
		},
		"",
	},
	// {client} placeholder in the middle of host
	{
		"http://api.{client}.com/",
		&URL{
			Scheme: "http",
			Host:   "api.{client}.com",
			Path:   "/",
		},
		"",
	},
	// {client} placeholder as full host with port
	{
		"http://{client}:8080/",
		&URL{
			Scheme: "http",
			Host:   "{client}:8080",
			Path:   "/",
		},
		"",
	},
	// {client} placeholder as subdomain with port
	{
		"http://{client}.example.com:8080/path",
		&URL{
			Scheme: "http",
			Host:   "{client}.example.com:8080",
			Path:   "/path",
		},
		"",
	},
	// {client} placeholder with path and query
	{
		"https://{client}.example.com/api?version=2",
		&URL{
			Scheme:   "https",
			Host:     "{client}.example.com",
			Path:     "/api",
			RawQuery: "version=2",
		},
		"",
	},
	{
		"//host.com",
		&URL{
			Host: "host.com",
		},
		"",
	},
}

// more useful string for debugging than fmt's struct printer
func ufmt(u *URL) string {
	var user, pass any
	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			pass = p
		}
	}
	return fmt.Sprintf("opaque=%q, scheme=%q, user=%#v, pass=%#v, host=%q, path=%q, rawpath=%q, rawq=%q, frag=%q, rawfrag=%q, forcequery=%v, omithost=%t",
		u.Opaque, u.Scheme, user, pass, u.Host, u.Path, u.RawPath, u.RawQuery, u.Fragment, u.RawFragment, u.ForceQuery, u.OmitHost)
}

func BenchmarkString(b *testing.B) {
	b.StopTimer()
	b.ReportAllocs()
	for _, tt := range urltests {
		u, err := Parse(tt.in)
		if err != nil {
			b.Errorf("Parse(%q) returned error %s", tt.in, err)
			continue
		}
		if tt.roundtrip == "" {
			continue
		}
		b.StartTimer()
		var g string
		for i := 0; i < b.N; i++ {
			g = URL_String(u)
		}
		b.StopTimer()
		if w := tt.roundtrip; b.N > 0 && g != w {
			b.Errorf("Parse(%q).String() == %q, want %q", tt.in, g, w)
		}
	}
}

func TestParse(t *testing.T) {
	for _, tt := range urltests {
		t.Run(tt.in, func(t *testing.T) {
			u, err := Parse(tt.in)
			require.NoError(t, err, "Parse(%q) returned unexpected error", tt.in)
			assert.True(t, reflect.DeepEqual(u, tt.out), "Parse(%q):\n\tgot  %v\n\twant %v\n", tt.in, ufmt(u), ufmt(tt.out))
		})
	}
}

const pathThatLooksSchemeRelative = "//not.a.user@not.a.host/just/a/path"

var parseRequestURLTests = []struct {
	url           string
	expectedValid bool
}{
	{"http://foo.com", true},
	{"http://foo.com/", true},
	{"http://foo.com/path", true},
	{"/", true},
	{pathThatLooksSchemeRelative, true},
	{"//not.a.user@%66%6f%6f.com/just/a/path/also", true},
	{"*", true},
	{"http://192.168.0.1/", true},
	{"http://192.168.0.1:8080/", true},
	{"http://[fe80::1]/", true},
	{"http://[fe80::1]:8080/", true},

	// Tests exercising RFC 6874 compliance:
	{"http://[fe80::1%25en0]/", true},                 // with alphanum zone identifier
	{"http://[fe80::1%25en0]:8080/", true},            // with alphanum zone identifier
	{"http://[fe80::1%25%65%6e%301-._~]/", true},      // with percent-encoded+unreserved zone identifier
	{"http://[fe80::1%25%65%6e%301-._~]:8080/", true}, // with percent-encoded+unreserved zone identifier

	// {client} placeholder in host
	{"http://{client}/", true},
	{"http://{client}.example.com/", true},
	{"http://api.{client}.com/", true},
	{"http://{client}:8080/", true},
	{"http://{client}.example.com:8080/path", true},
	{"https://{client}.example.com/api?version=2", true},

	// invalid placeholder formats in host
	{"http://{}/", false},                    // empty placeholder
	{"http://{{demo}.example.com/", false},   // double opening brace
	{"http://{demo.example.com/", false},     // unclosed placeholder
	{"http://demo}.example.com/", false},     // unmatched closing brace
	{"http://{ client}.example.com/", false}, // space inside placeholder
	{"http://{client}}.example.com/", false}, // extra closing brace after placeholder

	// placeholder is allowed in the path
	{"http://example.com/{client}/path", true}, // placeholder in path segment
	{"http://example.com/path/{client}", true}, // placeholder at end of path

	// placeholder is not allowed in the query
	{"http://example.com/?{client}=1", false}, // placeholder in query key
	{"http://example.com/?x={client}", false}, // placeholder in query value

	{"foo.html", false},
	{"../dir/", false},
	{" http://foo.com", false},
	{"http://192.168.0.%31/", false},
	{"http://192.168.0.%31:8080/", false},
	{"http://[fe80::%31]/", false},
	{"http://[fe80::%31]:8080/", false},
	{"http://[fe80::%31%25en0]/", false},
	{"http://[fe80::%31%25en0]:8080/", false},

	// These two cases are valid as textual representations as
	// described in RFC 4007, but are not valid as address
	// literals with IPv6 zone identifiers in URIs as described in
	// RFC 6874.
	{"http://[fe80::1%en0]/", false},
	{"http://[fe80::1%en0]:8080/", false},

	// Tests exercising RFC 3986 compliance
	{"https://[1:2:3:4:5:6:7:8]", true},             // full IPv6 address
	{"https://[2001:db8::a:b:c:d]", true},           // compressed IPv6 address
	{"https://[fe80::1%25eth0]", true},              // link-local address with zone ID (interface name)
	{"https://[fe80::abc:def%254]", true},           // link-local address with zone ID (interface index)
	{"https://[2001:db8::1]/path", true},            // compressed IPv6 address with path
	{"https://[fe80::1%25eth0]/path?query=1", true}, // link-local with zone, path, and query

	{"https://[::ffff:192.0.2.1]", true},
	{"https://[:1] ", false},
	{"https://[1:2:3:4:5:6:7:8:9]", false},
	{"https://[1::1::1]", false},
	{"https://[1:2:3:]", false},
	{"https://[ffff::127.0.0.4000]", false},
	{"https://[0:0::test.com]:80", false},
	{"https://[2001:db8::test.com]", false},
	{"https://[test.com]", false},
	{"https://1:2:3:4:5:6:7:8", false},
	{"https://1:2:3:4:5:6:7:8:80", false},
	{"https://example.com:80:", false},
}

func TestParseRequestURI(t *testing.T) {
	for _, test := range parseRequestURLTests {
		t.Run(test.url, func(t *testing.T) {
			_, err := ParseRequestURI(test.url)
			if test.expectedValid {
				assert.NoError(t, err, "ParseRequestURI(%q) should not error", test.url)
			} else {
				assert.Error(t, err, "ParseRequestURI(%q) should error", test.url)
			}
		})
	}

	url, err := ParseRequestURI(pathThatLooksSchemeRelative)
	require.NoError(t, err, "Unexpected error parsing %q", pathThatLooksSchemeRelative)
	assert.Equal(t, pathThatLooksSchemeRelative, url.Path, "ParseRequestURI path mismatch")
}

var stringURLTests = []struct {
	url  URL
	want string
}{
	// No leading slash on path should prepend slash on String() call
	{
		url: URL{
			Scheme: "http",
			Host:   "www.google.com",
			Path:   "search",
		},
		want: "http://www.google.com/search",
	},
	// Relative path with first element containing ":" should be prepended with "./", golang.org/issue/17184
	{
		url: URL{
			Path: "this:that",
		},
		want: "./this:that",
	},
	// Relative path with second element containing ":" should not be prepended with "./"
	{
		url: URL{
			Path: "here/this:that",
		},
		want: "here/this:that",
	},
	// Non-relative path with first element containing ":" should not be prepended with "./"
	{
		url: URL{
			Scheme: "http",
			Host:   "www.google.com",
			Path:   "this:that",
		},
		want: "http://www.google.com/this:that",
	},
}

func TestURLString(t *testing.T) {
	for _, tt := range urltests {
		t.Run(tt.in, func(t *testing.T) {
			u, err := Parse(tt.in)
			require.NoError(t, err, "Parse(%q) returned unexpected error", tt.in)
			expected := tt.in
			if tt.roundtrip != "" {
				expected = tt.roundtrip
			}
			s := URL_String(u)
			assert.Equal(t, expected, s, "URL string mismatch for %q", tt.in)
		})
	}

	for _, tt := range stringURLTests {
		t.Run(tt.want, func(t *testing.T) {
			got := URL_String(&tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestURLRedacted(t *testing.T) {
	cases := []struct {
		name string
		url  *URL
		want string
	}{
		{
			name: "non-blank Password",
			url: &URL{
				Scheme: "http",
				Host:   "host.tld",
				Path:   "this:that",
				User:   base_url.UserPassword("user", "password"),
			},
			want: "http://user:xxxxx@host.tld/this:that",
		},
		{
			name: "blank Password",
			url: &URL{
				Scheme: "http",
				Host:   "host.tld",
				Path:   "this:that",
				User:   base_url.User("user"),
			},
			want: "http://user@host.tld/this:that",
		},
		{
			name: "nil User",
			url: &URL{
				Scheme: "http",
				Host:   "host.tld",
				Path:   "this:that",
				User:   base_url.UserPassword("", "password"),
			},
			want: "http://:xxxxx@host.tld/this:that",
		},
		{
			name: "blank Username, blank Password",
			url: &URL{
				Scheme: "http",
				Host:   "host.tld",
				Path:   "this:that",
			},
			want: "http://host.tld/this:that",
		},
		{
			name: "empty URL",
			url:  &URL{},
			want: "",
		},
		{
			name: "nil URL",
			url:  nil,
			want: "",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Redacted(tt.url))
		})
	}
}

type EscapeTest struct {
	in  string
	out string
	err error
}

var unescapeTests = []EscapeTest{
	{
		"",
		"",
		nil,
	},
	{
		"abc",
		"abc",
		nil,
	},
	{
		"1%41",
		"1A",
		nil,
	},
	{
		"1%41%42%43",
		"1ABC",
		nil,
	},
	{
		"%4a",
		"J",
		nil,
	},
	{
		"%6F",
		"o",
		nil,
	},
	{
		"%", // not enough characters after %
		"",
		base_url.EscapeError("%"),
	},
	{
		"%a", // not enough characters after %
		"",
		base_url.EscapeError("%a"),
	},
	{
		"%1", // not enough characters after %
		"",
		base_url.EscapeError("%1"),
	},
	{
		"123%45%6", // not enough characters after %
		"",
		base_url.EscapeError("%6"),
	},
	{
		"%zzzzz", // invalid hex digits
		"",
		base_url.EscapeError("%zz"),
	},
	{
		"a+b",
		"a b",
		nil,
	},
	{
		"a%20b",
		"a b",
		nil,
	},
}

func TestUnescape(t *testing.T) {
	for _, tt := range unescapeTests {
		t.Run(tt.in, func(t *testing.T) {
			actual, err := QueryUnescape(tt.in)
			assert.Equal(t, tt.out, actual, "QueryUnescape(%q) mismatch", tt.in)
			if tt.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			in := tt.in
			out := tt.out
			if strings.Contains(tt.in, "+") {
				in = strings.ReplaceAll(tt.in, "+", "%20")
				actual, err := PathUnescape(in)
				assert.Equal(t, tt.out, actual, "PathUnescape(%q) mismatch", in)
				if tt.err != nil {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				if tt.err == nil {
					s, err := QueryUnescape(strings.ReplaceAll(tt.in, "+", "XXX"))
					if err != nil {
						return
					}
					in = tt.in
					out = strings.ReplaceAll(s, "XXX", "+")
				}
			}

			actual, err = PathUnescape(in)
			assert.Equal(t, out, actual, "PathUnescape(%q) mismatch", in)
			if tt.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

var queryEscapeTests = []EscapeTest{
	{
		"",
		"",
		nil,
	},
	{
		"abc",
		"abc",
		nil,
	},
	{
		"one two",
		"one+two",
		nil,
	},
	{
		"10%",
		"10%25",
		nil,
	},
	{
		" ?&=#+%!<>#\"{}|\\^[]`☺\t:/@$'()*,;",
		"+%3F%26%3D%23%2B%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09%3A%2F%40%24%27%28%29%2A%2C%3B",
		nil,
	},
}

func TestQueryEscape(t *testing.T) {
	for _, tt := range queryEscapeTests {
		t.Run(tt.in, func(t *testing.T) {
			actual := QueryEscape(tt.in)
			assert.Equal(t, tt.out, actual, "QueryEscape(%q) mismatch", tt.in)

			roundtrip, err := QueryUnescape(actual)
			assert.NoError(t, err, "QueryUnescape should not error")
			assert.Equal(t, tt.in, roundtrip, "escape:unescape should be identity")
		})
	}
}

var pathEscapeTests = []EscapeTest{
	{
		"",
		"",
		nil,
	},
	{
		"abc",
		"abc",
		nil,
	},
	{
		"abc+def",
		"abc+def",
		nil,
	},
	{
		"a/b",
		"a%2Fb",
		nil,
	},
	{
		"one two",
		"one%20two",
		nil,
	},
	{
		"10%",
		"10%25",
		nil,
	},
	{
		" ?&=#+%!<>#\"{}|\\^[]`☺\t:/@$'()*,;",
		"%20%3F&=%23+%25%21%3C%3E%23%22%7B%7D%7C%5C%5E%5B%5D%60%E2%98%BA%09:%2F@$%27%28%29%2A%2C%3B",
		nil,
	},
}

func TestPathEscape(t *testing.T) {
	for _, tt := range pathEscapeTests {
		t.Run(tt.in, func(t *testing.T) {
			actual := PathEscape(tt.in)
			assert.Equal(t, tt.out, actual, "PathEscape(%q) mismatch", tt.in)

			roundtrip, err := PathUnescape(actual)
			assert.NoError(t, err, "PathUnescape should not error")
			assert.Equal(t, tt.in, roundtrip, "escape:unescape should be identity")
		})
	}
}

type EncodeQueryTest struct {
	m        base_url.Values
	expected string
}

var encodeQueryTests = []EncodeQueryTest{
	{nil, ""},
	{base_url.Values{}, ""},
	{base_url.Values{"q": {"puppies"}, "oe": {"utf8"}}, "oe=utf8&q=puppies"},
	{base_url.Values{"q": {"dogs", "&", "7"}}, "q=dogs&q=%26&q=7"},
	{base_url.Values{
		"a": {"a1", "a2", "a3"},
		"b": {"b1", "b2", "b3"},
		"c": {"c1", "c2", "c3"},
	}, "a=a1&a=a2&a=a3&b=b1&b=b2&b=b3&c=c1&c=c2&c=c3"},
	{base_url.Values{
		"a": {"a"},
		"b": {"b"},
		"c": {"c"},
		"d": {"d"},
		"e": {"e"},
		"f": {"f"},
		"g": {"g"},
		"h": {"h"},
		"i": {"i"},
	}, "a=a&b=b&c=c&d=d&e=e&f=f&g=g&h=h&i=i"},
}

func TestEncodeQuery(t *testing.T) {
	for _, tt := range encodeQueryTests {
		t.Run(tt.expected, func(t *testing.T) {
			q := tt.m.Encode()
			assert.Equal(t, tt.expected, q)
		})
	}
}

func BenchmarkEncodeQuery(b *testing.B) {
	for _, tt := range encodeQueryTests {
		b.Run(tt.expected, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tt.m.Encode()
			}
		})
	}
}

var resolvePathTests = []struct {
	base, ref, expected string
}{
	{"a/b", ".", "/a/"},
	{"a/b", "c", "/a/c"},
	{"a/b", "..", "/"},
	{"a/", "..", "/"},
	{"a/", "../..", "/"},
	{"a/b/c", "..", "/a/"},
	{"a/b/c", "../d", "/a/d"},
	{"a/b/c", ".././d", "/a/d"},
	{"a/b", "./..", "/"},
	{"a/./b", ".", "/a/"},
	{"a/../", ".", "/"},
	{"a/.././b", "c", "/c"},
}

func TestResolvePath(t *testing.T) {
	for _, test := range resolvePathTests {
		t.Run(test.base+" "+test.ref, func(t *testing.T) {
			got := resolvePath(test.base, test.ref)
			assert.Equal(t, test.expected, got, "For %q + %q", test.base, test.ref)
		})
	}
}

func BenchmarkResolvePath(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resolvePath("a/b/c", ".././d")
	}
}

var resolveReferenceTests = []struct {
	base, rel, expected string
}{
	// Absolute URL references
	{"http://foo.com?a=b", "https://bar.com/", "https://bar.com/"},
	{"http://foo.com/", "https://bar.com/?a=b", "https://bar.com/?a=b"},
	{"http://foo.com/", "https://bar.com/?", "https://bar.com/?"},
	{"http://foo.com/bar", "mailto:foo@example.com", "mailto:foo@example.com"},

	// Path-absolute references
	{"http://foo.com/bar", "/baz", "http://foo.com/baz"},
	{"http://foo.com/bar?a=b#f", "/baz", "http://foo.com/baz"},
	{"http://foo.com/bar?a=b", "/baz?", "http://foo.com/baz?"},
	{"http://foo.com/bar?a=b", "/baz?c=d", "http://foo.com/baz?c=d"},

	// Multiple slashes
	{"http://foo.com/bar", "http://foo.com//baz", "http://foo.com//baz"},
	{"http://foo.com/bar", "http://foo.com///baz/quux", "http://foo.com///baz/quux"},

	// Scheme-relative
	{"https://foo.com/bar?a=b", "//bar.com/quux", "https://bar.com/quux"},

	// Path-relative references:

	// ... current directory
	{"http://foo.com", ".", "http://foo.com/"},
	{"http://foo.com/bar", ".", "http://foo.com/"},
	{"http://foo.com/bar/", ".", "http://foo.com/bar/"},

	// ... going down
	{"http://foo.com", "bar", "http://foo.com/bar"},
	{"http://foo.com/", "bar", "http://foo.com/bar"},
	{"http://foo.com/bar/baz", "quux", "http://foo.com/bar/quux"},

	// ... going up
	{"http://foo.com/bar/baz", "../quux", "http://foo.com/quux"},
	{"http://foo.com/bar/baz", "../../../../../quux", "http://foo.com/quux"},
	{"http://foo.com/bar", "..", "http://foo.com/"},
	{"http://foo.com/bar/baz", "./..", "http://foo.com/"},
	// ".." in the middle (issue 3560)
	{"http://foo.com/bar/baz", "quux/dotdot/../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/.././tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/./../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/././../../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/./.././../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/dotdot/./../../.././././tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/../dotdot/../dot/./tail/..", "http://foo.com/bar/quux/dot/"},

	// Remove any dot-segments prior to forming the target URI.
	// https://datatracker.ietf.org/doc/html/rfc3986#section-5.2.4
	{"http://foo.com/dot/./dotdot/../foo/bar", "../baz", "http://foo.com/dot/baz"},

	// Triple dot isn't special
	{"http://foo.com/bar", "...", "http://foo.com/..."},

	// Fragment
	{"http://foo.com/bar", ".#frag", "http://foo.com/#frag"},
	{"http://example.org/", "#!$&%27()*+,;=", "http://example.org/#!$&%27()*+,;="},

	// Paths with escaping (issue 16947).
	{"http://foo.com/foo%2fbar/", "../baz", "http://foo.com/baz"},
	{"http://foo.com/1/2%2f/3%2f4/5", "../../a/b/c", "http://foo.com/1/a/b/c"},
	{"http://foo.com/1/2/3", "./a%2f../../b/..%2fc", "http://foo.com/1/2/b/..%2fc"},
	{"http://foo.com/1/2%2f/3%2f4/5", "./a%2f../b/../c", "http://foo.com/1/2%2f/3%2f4/a%2f../c"},
	{"http://foo.com/foo%20bar/", "../baz", "http://foo.com/baz"},
	{"http://foo.com/foo", "../bar%2fbaz", "http://foo.com/bar%2fbaz"},
	{"http://foo.com/foo%2dbar/", "./baz-quux", "http://foo.com/foo%2dbar/baz-quux"},

	// RFC 3986: Normal Examples
	// https://datatracker.ietf.org/doc/html/rfc3986#section-5.4.1
	{"http://a/b/c/d;p?q", "g:h", "g:h"},
	{"http://a/b/c/d;p?q", "g", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "./g", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g/", "http://a/b/c/g/"},
	{"http://a/b/c/d;p?q", "/g", "http://a/g"},
	{"http://a/b/c/d;p?q", "//g", "http://g"},
	{"http://a/b/c/d;p?q", "?y", "http://a/b/c/d;p?y"},
	{"http://a/b/c/d;p?q", "g?y", "http://a/b/c/g?y"},
	{"http://a/b/c/d;p?q", "#s", "http://a/b/c/d;p?q#s"},
	{"http://a/b/c/d;p?q", "g#s", "http://a/b/c/g#s"},
	{"http://a/b/c/d;p?q", "g?y#s", "http://a/b/c/g?y#s"},
	{"http://a/b/c/d;p?q", ";x", "http://a/b/c/;x"},
	{"http://a/b/c/d;p?q", "g;x", "http://a/b/c/g;x"},
	{"http://a/b/c/d;p?q", "g;x?y#s", "http://a/b/c/g;x?y#s"},
	{"http://a/b/c/d;p?q", "", "http://a/b/c/d;p?q"},
	{"http://a/b/c/d;p?q", ".", "http://a/b/c/"},
	{"http://a/b/c/d;p?q", "./", "http://a/b/c/"},
	{"http://a/b/c/d;p?q", "..", "http://a/b/"},
	{"http://a/b/c/d;p?q", "../", "http://a/b/"},
	{"http://a/b/c/d;p?q", "../g", "http://a/b/g"},
	{"http://a/b/c/d;p?q", "../..", "http://a/"},
	{"http://a/b/c/d;p?q", "../../", "http://a/"},
	{"http://a/b/c/d;p?q", "../../g", "http://a/g"},

	// RFC 3986: Abnormal Examples
	// https://datatracker.ietf.org/doc/html/rfc3986#section-5.4.2
	{"http://a/b/c/d;p?q", "../../../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "../../../../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "/./g", "http://a/g"},
	{"http://a/b/c/d;p?q", "/../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "g.", "http://a/b/c/g."},
	{"http://a/b/c/d;p?q", ".g", "http://a/b/c/.g"},
	{"http://a/b/c/d;p?q", "g..", "http://a/b/c/g.."},
	{"http://a/b/c/d;p?q", "..g", "http://a/b/c/..g"},
	{"http://a/b/c/d;p?q", "./../g", "http://a/b/g"},
	{"http://a/b/c/d;p?q", "./g/.", "http://a/b/c/g/"},
	{"http://a/b/c/d;p?q", "g/./h", "http://a/b/c/g/h"},
	{"http://a/b/c/d;p?q", "g/../h", "http://a/b/c/h"},
	{"http://a/b/c/d;p?q", "g;x=1/./y", "http://a/b/c/g;x=1/y"},
	{"http://a/b/c/d;p?q", "g;x=1/../y", "http://a/b/c/y"},
	{"http://a/b/c/d;p?q", "g?y/./x", "http://a/b/c/g?y/./x"},
	{"http://a/b/c/d;p?q", "g?y/../x", "http://a/b/c/g?y/../x"},
	{"http://a/b/c/d;p?q", "g#s/./x", "http://a/b/c/g#s/./x"},
	{"http://a/b/c/d;p?q", "g#s/../x", "http://a/b/c/g#s/../x"},

	// Extras.
	{"https://a/b/c/d;p?q", "//g?q", "https://g?q"},
	{"https://a/b/c/d;p?q", "//g#s", "https://g#s"},
	{"https://a/b/c/d;p?q", "//g/d/e/f?y#s", "https://g/d/e/f?y#s"},
	{"https://a/b/c/d;p#s", "?y", "https://a/b/c/d;p?y"},
	{"https://a/b/c/d;p?q#s", "?y", "https://a/b/c/d;p?y"},

	// Empty path and query but with ForceQuery (issue 46033).
	{"https://a/b/c/d;p?q#s", "?", "https://a/b/c/d;p?"},

	// Opaque URLs (issue 66084).
	{"https://foo.com/bar?a=b", "http:opaque", "http:opaque"},
	{"http:opaque?x=y#zzz", "https:/foo?a=b#frag", "https:/foo?a=b#frag"},
	{"http:opaque?x=y#zzz", "https:foo:bar", "https:foo:bar"},
	{"http:opaque?x=y#zzz", "https:bar/baz?a=b#frag", "https:bar/baz?a=b#frag"},
	{"http:opaque?x=y#zzz", "https://user@host:1234?a=b#frag", "https://user@host:1234?a=b#frag"},
	{"http:opaque?x=y#zzz", "?a=b#frag", "http:opaque?a=b#frag"},
}

func TestResolveReference(t *testing.T) {
	mustParse := func(url string) *URL {
		u, err := Parse(url)
		require.NoError(t, err, "Parse(%q) got err", url)
		return u
	}
	opaque := &URL{Scheme: "scheme", Opaque: "opaque"}
	for _, test := range resolveReferenceTests {
		t.Run(test.base+" "+test.rel, func(t *testing.T) {
			base := mustParse(test.base)
			rel := mustParse(test.rel)
			url := URL_ResolveReference(base, rel)
			got := URL_String(url)
			assert.Equal(t, test.expected, got, "URL(%q).ResolveReference(%q)", test.base, test.rel)
			assert.NotSame(t, base, url, "Expected URL_ResolveReference to return new URL instance")

			url, err := URL_Parse(base, test.rel)
			require.NoError(t, err, "URL(%q).Parse(%q) failed", test.base, test.rel)
			got = URL_String(url)
			assert.Equal(t, test.expected, got, "URL(%q).Parse(%q)", test.base, test.rel)
			assert.NotSame(t, base, url, "Expected URL_Parse to return new URL instance")

			url = URL_ResolveReference(base, opaque)
			assert.Equal(t, opaque, url, "ResolveReference failed to resolve opaque URL")

			url, err = URL_Parse(base, "scheme:opaque")
			require.NoError(t, err, "URL(%q).Parse(scheme:opaque) failed", test.base)
			assert.Equal(t, opaque, url, "Parse failed to resolve opaque URL")
			assert.NotSame(t, base, url, "Expected URL.Parse to return new URL instance")
		})
	}
}

func TestQueryValues(t *testing.T) {
	u, _ := Parse("http://x.com?foo=bar&bar=1&bar=2&baz")
	v := URL_Query(u)
	assert.Len(t, v, 3, "Expected 3 keys in Query values")
	assert.Equal(t, "bar", v.Get("foo"))
	assert.Equal(t, "", v.Get("Foo"), "Query should be case sensitive")
	assert.Equal(t, "1", v.Get("bar"))
	assert.Equal(t, "", v.Get("baz"))
	assert.True(t, v.Has("foo"))
	assert.True(t, v.Has("bar"))
	assert.True(t, v.Has("baz"))
	assert.False(t, v.Has("noexist"))
	v.Del("bar")
	assert.Equal(t, "", v.Get("bar"), "bar should be deleted")
}

type parseTest struct {
	query string
	out   base_url.Values
	ok    bool
}

var parseTests = []parseTest{
	{
		query: "a=1",
		out:   base_url.Values{"a": []string{"1"}},
		ok:    true,
	},
	{
		query: "a=1&b=2",
		out:   base_url.Values{"a": []string{"1"}, "b": []string{"2"}},
		ok:    true,
	},
	{
		query: "a=1&a=2&a=banana",
		out:   base_url.Values{"a": []string{"1", "2", "banana"}},
		ok:    true,
	},
	{
		query: "ascii=%3Ckey%3A+0x90%3E",
		out:   base_url.Values{"ascii": []string{"<key: 0x90>"}},
		ok:    true,
	},
	{
		query: "a=1;b=2",
		out:   base_url.Values{},
		ok:    false,
	},
	{
		query: "a;b=1",
		out:   base_url.Values{},
		ok:    false,
	},
	{
		query: "a=%3B", // hex encoding for semicolon
		out:   base_url.Values{"a": []string{";"}},
		ok:    true,
	},
	{
		query: "a%3Bb=1",
		out:   base_url.Values{"a;b": []string{"1"}},
		ok:    true,
	},
	{
		query: "a=1&a=2;a=banana",
		out:   base_url.Values{"a": []string{"1"}},
		ok:    false,
	},
	{
		query: "a;b&c=1",
		out:   base_url.Values{"c": []string{"1"}},
		ok:    false,
	},
	{
		query: "a=1&b=2;a=3&c=4",
		out:   base_url.Values{"a": []string{"1"}, "c": []string{"4"}},
		ok:    false,
	},
	{
		query: "a=1&b=2;c=3",
		out:   base_url.Values{"a": []string{"1"}},
		ok:    false,
	},
	{
		query: ";",
		out:   base_url.Values{},
		ok:    false,
	},
	{
		query: "a=1;",
		out:   base_url.Values{},
		ok:    false,
	},
	{
		query: "a=1&;",
		out:   base_url.Values{"a": []string{"1"}},
		ok:    false,
	},
	{
		query: ";a=1&b=2",
		out:   base_url.Values{"b": []string{"2"}},
		ok:    false,
	},
	{
		query: "a=1&b=2;",
		out:   base_url.Values{"a": []string{"1"}},
		ok:    false,
	},
}

func TestParseQuery(t *testing.T) {
	for _, test := range parseTests {
		t.Run(test.query, func(t *testing.T) {
			form, err := ParseQuery(test.query)
			if test.ok {
				assert.NoError(t, err, "ParseQuery should not error for %q", test.query)
			} else {
				assert.Error(t, err, "ParseQuery should error for %q", test.query)
			}
			assert.Len(t, form, len(test.out), "form length mismatch for %q", test.query)
			for k, evs := range test.out {
				vs, ok := form[k]
				assert.True(t, ok, "Missing key %q in form", k)
				assert.Len(t, vs, len(evs), "len(form[%q]) mismatch", k)
				for j, ev := range evs {
					assert.Equal(t, ev, vs[j], "form[%q][%d] mismatch", k, j)
				}
			}
		})
	}
}

type RequestURITest struct {
	url *URL
	out string
}

var requritests = []RequestURITest{
	{
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Path:   "",
		},
		"/",
	},
	{
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Path:   "/a b",
		},
		"/a%20b",
	},
	// golang.org/issue/4860 variant 1
	{
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Opaque: "/%2F/%2F/",
		},
		"/%2F/%2F/",
	},
	// golang.org/issue/4860 variant 2
	{
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Opaque: "//other.example.com/%2F/%2F/",
		},
		"http://other.example.com/%2F/%2F/",
	},
	// better fix for issue 4860
	{
		&URL{
			Scheme:  "http",
			Host:    "example.com",
			Path:    "/////",
			RawPath: "/%2F/%2F/",
		},
		"/%2F/%2F/",
	},
	{
		&URL{
			Scheme:  "http",
			Host:    "example.com",
			Path:    "/////",
			RawPath: "/WRONG/", // ignored because doesn't match Path
		},
		"/////",
	},
	{
		&URL{
			Scheme:   "http",
			Host:     "example.com",
			Path:     "/a b",
			RawQuery: "q=go+language",
		},
		"/a%20b?q=go+language",
	},
	{
		&URL{
			Scheme:   "http",
			Host:     "example.com",
			Path:     "/a b",
			RawPath:  "/a b", // ignored because invalid
			RawQuery: "q=go+language",
		},
		"/a%20b?q=go+language",
	},
	{
		&URL{
			Scheme:   "http",
			Host:     "example.com",
			Path:     "/a?b",
			RawPath:  "/a?b", // ignored because invalid
			RawQuery: "q=go+language",
		},
		"/a%3Fb?q=go+language",
	},
	{
		&URL{
			Scheme: "myschema",
			Opaque: "opaque",
		},
		"opaque",
	},
	{
		&URL{
			Scheme:   "myschema",
			Opaque:   "opaque",
			RawQuery: "q=go+language",
		},
		"opaque?q=go+language",
	},
	{
		&URL{
			Scheme: "http",
			Host:   "example.com",
			Path:   "//foo",
		},
		"//foo",
	},
	{
		&URL{
			Scheme:     "http",
			Host:       "example.com",
			Path:       "/foo",
			ForceQuery: true,
		},
		"/foo?",
	},
}

func TestRequestURI(t *testing.T) {
	for _, tt := range requritests {
		t.Run(tt.out, func(t *testing.T) {
			s := URL_RequestURI(tt.url)
			assert.Equal(t, tt.out, s)
		})
	}
}

func TestParseFailure(t *testing.T) {
	const url = "%gh&%ij"
	_, err := ParseQuery(url)
	errStr := fmt.Sprint(err)
	assert.Contains(t, errStr, "%gh", "ParseQuery error should contain %%gh")
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"http://[::1]", false},
		{"http://[::1]:80", false},
		{"http://[::1]:namedport", true}, // rfc3986 3.2.3
		{"http://x:namedport", true},     // rfc3986 3.2.3
		{"http://[::1]/", false},
		{"http://[::1]a", true},
		{"http://[::1]%23", true},
		{"http://[::1%25en0]", false},    // valid zone id
		{"http://[::1]:", false},         // colon, but no port OK
		{"http://x:", false},             // colon, but no port OK
		{"http://[::1]:%38%30", true},    // not allowed: % encoding only for non-ASCII
		{"http://[::1%25%41]", false},    // RFC 6874 allows over-escaping in zone
		{"http://[%10::1]", true},        // no %xx escapes in IP address
		{"http://[::1]/%48", false},      // %xx in path is fine
		{"http://%41:8080/", true},       // not allowed: % encoding only for non-ASCII
		{"mysql://x@y(z:123)/foo", true}, // not well-formed per RFC 3986, golang.org/issue/33646
		{"mysql://x@y(1.2.3.4:123)/foo", true},

		{" http://foo.com", true},  // invalid character in schema
		{"ht tp://foo.com", true},  // invalid character in schema
		{"ahttp://foo.com", false}, // valid schema characters
		{"1http://foo.com", true},  // invalid character in schema

		{"http://[]%20%48%54%54%50%2f%31%2e%31%0a%4d%79%48%65%61%64%65%72%3a%20%31%32%33%0a%0a/", true}, // golang.org/issue/11208
		{"http://a b.com/", true},    // no space in host name please
		{"cache_object://foo", true}, // scheme cannot have _, relative path cannot have : in first segment
		{"cache_object:foo", true},
		{"cache_object:foo/bar", true},
		{"cache_object/:foo/bar", false},

		{"http://[192.168.0.1]/", true},              // IPv4 in brackets
		{"http://[192.168.0.1]:8080/", true},         // IPv4 in brackets with port
		{"http://[::ffff:192.168.0.1]/", false},      // IPv4-mapped IPv6 in brackets
		{"http://[::ffff:192.168.0.1000]/", true},    // Out of range IPv4-mapped IPv6 in brackets
		{"http://[::ffff:192.168.0.1]:8080/", false}, // IPv4-mapped IPv6 in brackets with port
		{"http://[::ffff:c0a8:1]/", false},           // IPv4-mapped IPv6 in brackets (hex)
		{"http://[not-an-ip]/", true},                // invalid IP string in brackets
		{"http://[fe80::1%foo]/", true},              // invalid zone format in brackets
		{"http://[fe80::1", true},                    // missing closing bracket
		{"http://fe80::1]/", true},                   // missing opening bracket
		{"http://[test.com]/", true},                 // domain name in brackets
		{"http://example.com[::1]", true},            // IPv6 literal doesn't start with '['
		{"http://example.com[::1", true},
		{"http://[::1", true},
		{"http://.[::1]", true},
		{"http:// [::1]", true},
		{"hxxp://mathepqo[.]serveftp(.)com:9059", true},

		// {client} placeholder — must not produce an error
		{"http://{client}/", false},
		{"http://{client}.example.com/", false},
		{"http://api.{client}.com/", false},
		{"http://{client}:8080/", false},
		{"http://{client}.example.com:8080/path", false},

		// invalid placeholder formats in host — must produce an error
		{"http://{}/", true},                    // empty placeholder
		{"http://{{demo}.example.com/", true},   // double opening brace
		{"http://{demo.example.com/", true},     // unclosed placeholder
		{"http://demo}.example.com/", true},     // unmatched closing brace
		{"http://{ client}.example.com/", true}, // space inside placeholder
		{"http://{client}}.example.com/", true}, // extra closing brace after placeholder

		// placeholder in the path — must not produce an error
		{"http://example.com/{client}/path", false}, // placeholder in path segment
		{"http://example.com/path/{client}", false}, // placeholder at end of path

		// placeholder in the query — must produce an error
		{"http://example.com/?{client}=1", true}, // placeholder in query key
		{"http://example.com/?x={client}", true}, // placeholder in query value
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			u, err := Parse(tt.in)
			if tt.wantErr {
				assert.Error(t, err, "Parse(%q) should error", tt.in)
			} else {
				assert.NoError(t, err, "Parse(%q) should not error", tt.in)
				assert.NotNil(t, u)
			}
		})
	}
}

// Issue 11202
func TestStarRequest(t *testing.T) {
	u, err := Parse("*")
	require.NoError(t, err)
	got := URL_RequestURI(u)
	assert.Equal(t, "*", got)
}

type shouldEscapeTest struct {
	in     byte
	mode   encoding
	escape bool
}

var shouldEscapeTests = []shouldEscapeTest{
	// Unreserved characters (§2.3)
	{'a', encodePath, false},
	{'a', encodeUserPassword, false},
	{'a', encodeQueryComponent, false},
	{'a', encodeFragment, false},
	{'a', encodeHost, false},
	{'z', encodePath, false},
	{'A', encodePath, false},
	{'Z', encodePath, false},
	{'0', encodePath, false},
	{'9', encodePath, false},
	{'-', encodePath, false},
	{'-', encodeUserPassword, false},
	{'-', encodeQueryComponent, false},
	{'-', encodeFragment, false},
	{'.', encodePath, false},
	{'_', encodePath, false},
	{'~', encodePath, false},

	// User information (§3.2.1)
	{':', encodeUserPassword, true},
	{'/', encodeUserPassword, true},
	{'?', encodeUserPassword, true},
	{'@', encodeUserPassword, true},
	{'$', encodeUserPassword, false},
	{'&', encodeUserPassword, false},
	{'+', encodeUserPassword, false},
	{',', encodeUserPassword, false},
	{';', encodeUserPassword, false},
	{'=', encodeUserPassword, false},

	// Host (IP address, IPv6 address, registered name, port suffix; §3.2.2)
	{'!', encodeHost, false},
	{'$', encodeHost, false},
	{'&', encodeHost, false},
	{'\'', encodeHost, false},
	{'(', encodeHost, false},
	{')', encodeHost, false},
	{'*', encodeHost, false},
	{'+', encodeHost, false},
	{',', encodeHost, false},
	{';', encodeHost, false},
	{'=', encodeHost, false},
	{':', encodeHost, false},
	{'[', encodeHost, false},
	{']', encodeHost, false},
	{'0', encodeHost, false},
	{'9', encodeHost, false},
	{'A', encodeHost, false},
	{'z', encodeHost, false},
	{'_', encodeHost, false},
	{'-', encodeHost, false},
	{'.', encodeHost, false},
}

func TestShouldEscape(t *testing.T) {
	for _, tt := range shouldEscapeTests {
		t.Run(fmt.Sprintf("%q_%v", tt.in, tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.escape, shouldEscape(tt.in, tt.mode), "shouldEscape(%q, %v) mismatch", tt.in, tt.mode)
		})
	}
}

type timeoutError struct {
	timeout bool
}

func (e *timeoutError) Error() string { return "timeout error" }
func (e *timeoutError) Timeout() bool { return e.timeout }

type temporaryError struct {
	temporary bool
}

func (e *temporaryError) Error() string   { return "temporary error" }
func (e *temporaryError) Temporary() bool { return e.temporary }

type timeoutTemporaryError struct {
	timeoutError
	temporaryError
}

func (e *timeoutTemporaryError) Error() string { return "timeout/temporary error" }

var netErrorTests = []struct {
	err       error
	timeout   bool
	temporary bool
}{{
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutError{timeout: true}},
	timeout:   true,
	temporary: false,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutError{timeout: false}},
	timeout:   false,
	temporary: false,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &temporaryError{temporary: true}},
	timeout:   false,
	temporary: true,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &temporaryError{temporary: false}},
	timeout:   false,
	temporary: false,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutTemporaryError{timeoutError{timeout: true}, temporaryError{temporary: true}}},
	timeout:   true,
	temporary: true,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutTemporaryError{timeoutError{timeout: false}, temporaryError{temporary: true}}},
	timeout:   false,
	temporary: true,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutTemporaryError{timeoutError{timeout: true}, temporaryError{temporary: false}}},
	timeout:   true,
	temporary: false,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: &timeoutTemporaryError{timeoutError{timeout: false}, temporaryError{temporary: false}}},
	timeout:   false,
	temporary: false,
}, {
	err:       &base_url.Error{Op: "Get", URL: "http://google.com/", Err: io.EOF},
	timeout:   false,
	temporary: false,
}}

// Test that base_url.Error implements net.Error and that it forwards
func TestURLErrorImplementsNetError(t *testing.T) {
	for i, tt := range netErrorTests {
		t.Run(fmt.Sprintf("case_%d", i+1), func(t *testing.T) {
			err, ok := tt.err.(net.Error)
			assert.True(t, ok, "%d: %T should implement net.Error", i+1, tt.err)
			assert.Equal(t, tt.timeout, err.Timeout(), "%d: err.Timeout() mismatch", i+1)
			assert.Equal(t, tt.temporary, err.Temporary(), "%d: err.Temporary() mismatch", i+1)
		})
	}
}

func TestURLHostnameAndPort(t *testing.T) {
	tests := []struct {
		in   string // URL.Host field
		host string
		port string
	}{
		{"foo.com:80", "foo.com", "80"},
		{"foo.com", "foo.com", ""},
		{"foo.com:", "foo.com", ""},
		{"FOO.COM", "FOO.COM", ""}, // no canonicalization
		{"1.2.3.4", "1.2.3.4", ""},
		{"1.2.3.4:80", "1.2.3.4", "80"},
		{"[1:2:3:4]", "1:2:3:4", ""},
		{"[1:2:3:4]:80", "1:2:3:4", "80"},
		{"[::1]:80", "::1", "80"},
		{"[::1]", "::1", ""},
		{"[::1]:", "::1", ""},
		{"localhost", "localhost", ""},
		{"localhost:443", "localhost", "443"},
		{"some.super.long.domain.example.org:8080", "some.super.long.domain.example.org", "8080"},
		{"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:17000", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "17000"},
		{"[2001:0db8:85a3:0000:0000:8a2e:0370:7334]", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", ""},

		// Ensure that even when not valid, Host is one of "Hostname",
		// "Hostname:Port", "[Hostname]" or "[Hostname]:Port".
		// See https://golang.org/issue/29098.
		{"[google.com]:80", "google.com", "80"},
		{"google.com]:80", "google.com]", "80"},
		{"google.com:80_invalid_port", "google.com:80_invalid_port", ""},
		{"[::1]extra]:80", "::1]extra", "80"},
		{"google.com]extra:extra", "google.com]extra:extra", ""},

		// {client} placeholder in host
		{"{client}", "{client}", ""},
		{"{client}:8080", "{client}", "8080"},
		{"{client}.example.com", "{client}.example.com", ""},
		{"{client}.example.com:443", "{client}.example.com", "443"},
		{"api.{client}.com", "api.{client}.com", ""},
		{"api.{client}.com:9090", "api.{client}.com", "9090"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			u := &URL{Host: tt.in}
			host, port := URL_Hostname(u), URL_Port(u)
			assert.Equal(t, tt.host, host, "Hostname for Host %q mismatch", tt.in)
			assert.Equal(t, tt.port, port, "Port for Host %q mismatch", tt.in)
		})
	}
}

var (
	_ encodingPkg.BinaryMarshaler   = (*URLT)(nil)
	_ encodingPkg.BinaryUnmarshaler = (*URLT)(nil)
	_ encodingPkg.BinaryAppender    = (*URLT)(nil)
)

func TestJSON(t *testing.T) {
	u, err := Parse("https://www.google.com/x?y=z")
	require.NoError(t, err)
	js, err := json.Marshal(u)
	require.NoError(t, err)

	u1 := new(URLT)
	err = json.Unmarshal(js, u1)
	require.NoError(t, err)
	assert.Equal(t, URL_String(u), URL_String((*URL)(u1)))
}

func TestGob(t *testing.T) {
	u, err := Parse("https://www.google.com/x?y=z")
	require.NoError(t, err)
	var w bytes.Buffer
	err = gob.NewEncoder(&w).Encode(u)
	require.NoError(t, err)

	u1 := new(URLT)
	err = gob.NewDecoder(&w).Decode(u1)
	require.NoError(t, err)
	assert.Equal(t, URL_String(u), URL_String((*URL)(u1)))
}

func TestNilUser(t *testing.T) {
	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("unexpected panic: %v", v)
		}
	}()

	u, err := Parse("http://foo.com/")
	require.NoError(t, err)

	v := u.User.Username()
	assert.Empty(t, v, "expected empty username")

	v, ok := u.User.Password()
	assert.Empty(t, v, "expected empty password")
	assert.False(t, ok)

	v = u.User.String()
	assert.Empty(t, v, "expected empty string")
}

func TestInvalidUserPassword(t *testing.T) {
	_, err := Parse("http://user^:passwo^rd@foo.com/")
	assert.Error(t, err)
	assert.Contains(t, fmt.Sprint(err), "net/url: invalid userinfo")
}

func TestRejectControlCharacters(t *testing.T) {
	tests := []string{
		"http://foo.com/?foo\nbar",
		"http\r://foo.com/",
		"http://foo\x7f.com/",
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			_, err := Parse(s)
			const wantSub = "net/url: invalid control character in URL"
			got := fmt.Sprint(err)
			assert.Contains(t, got, wantSub)
		})
	}

	_, err := Parse("http://foo.com/ctl\x80")
	assert.NoError(t, err, "should not reject non-ASCII control byte")
}

var escapeBenchmarks = []struct {
	unescaped string
	query     string
	path      string
}{
	{
		unescaped: "one two",
		query:     "one+two",
		path:      "one%20two",
	},
	{
		unescaped: "Фотки собак",
		query:     "%D0%A4%D0%BE%D1%82%D0%BA%D0%B8+%D1%81%D0%BE%D0%B1%D0%B0%D0%BA",
		path:      "%D0%A4%D0%BE%D1%82%D0%BA%D0%B8%20%D1%81%D0%BE%D0%B1%D0%B0%D0%BA",
	},

	{
		unescaped: "shortrun(break)shortrun",
		query:     "shortrun%28break%29shortrun",
		path:      "shortrun%28break%29shortrun",
	},

	{
		unescaped: "longerrunofcharacters(break)anotherlongerrunofcharacters",
		query:     "longerrunofcharacters%28break%29anotherlongerrunofcharacters",
		path:      "longerrunofcharacters%28break%29anotherlongerrunofcharacters",
	},

	{
		unescaped: strings.Repeat("padded/with+various%characters?that=need$some@escaping+paddedsowebreak/256bytes", 4),
		query:     strings.Repeat("padded%2Fwith%2Bvarious%25characters%3Fthat%3Dneed%24some%40escaping%2Bpaddedsowebreak%2F256bytes", 4),
		path:      strings.Repeat("padded%2Fwith+various%25characters%3Fthat=need$some@escaping+paddedsowebreak%2F256bytes", 4),
	},
}

func BenchmarkQueryEscape(b *testing.B) {
	for _, tc := range escapeBenchmarks {
		b.Run("", func(b *testing.B) {
			b.ReportAllocs()
			var g string
			for i := 0; i < b.N; i++ {
				g = QueryEscape(tc.unescaped)
			}
			b.StopTimer()
			if g != tc.query {
				b.Errorf("QueryEscape(%q) == %q, want %q", tc.unescaped, g, tc.query)
			}
		})
	}
}

func BenchmarkPathEscape(b *testing.B) {
	for _, tc := range escapeBenchmarks {
		b.Run("", func(b *testing.B) {
			b.ReportAllocs()
			var g string
			for i := 0; i < b.N; i++ {
				g = PathEscape(tc.unescaped)
			}
			b.StopTimer()
			if g != tc.path {
				b.Errorf("PathEscape(%q) == %q, want %q", tc.unescaped, g, tc.path)
			}
		})
	}
}

func BenchmarkQueryUnescape(b *testing.B) {
	for _, tc := range escapeBenchmarks {
		b.Run("", func(b *testing.B) {
			b.ReportAllocs()
			var g string
			for i := 0; i < b.N; i++ {
				g, _ = QueryUnescape(tc.query)
			}
			b.StopTimer()
			if g != tc.unescaped {
				b.Errorf("QueryUnescape(%q) == %q, want %q", tc.query, g, tc.unescaped)
			}
		})
	}
}

func BenchmarkPathUnescape(b *testing.B) {
	for _, tc := range escapeBenchmarks {
		b.Run("", func(b *testing.B) {
			b.ReportAllocs()
			var g string
			for i := 0; i < b.N; i++ {
				g, _ = PathUnescape(tc.path)
			}
			b.StopTimer()
			if g != tc.unescaped {
				b.Errorf("PathUnescape(%q) == %q, want %q", tc.path, g, tc.unescaped)
			}
		})
	}
}

func TestJoinPath(t *testing.T) { //NOSONAR
	tests := []struct {
		base string
		elem []string
		out  string
	}{
		{
			base: "https://go.googlesource.com",
			elem: []string{"go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com/a/b/c",
			elem: []string{"../../../go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com/",
			elem: []string{"../go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com",
			elem: []string{"../go", "../../go", "../../../go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com/../go",
			elem: nil,
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com/",
			elem: []string{"./go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com//",
			elem: []string{"/go"},
			out:  "https://go.googlesource.com/go",
		},
		{
			base: "https://go.googlesource.com//",
			elem: []string{"/go", "a", "b", "c"},
			out:  "https://go.googlesource.com/go/a/b/c",
		},
		{
			base: "http://[fe80::1%en0]:8080/",
			elem: []string{"/go"},
		},
		{
			base: "https://go.googlesource.com",
			elem: []string{"go/"},
			out:  "https://go.googlesource.com/go/",
		},
		{
			base: "https://go.googlesource.com",
			elem: []string{"go//"},
			out:  "https://go.googlesource.com/go/",
		},
		{
			base: "https://go.googlesource.com",
			elem: nil,
			out:  "https://go.googlesource.com/",
		},
		{
			base: "https://go.googlesource.com/",
			elem: nil,
			out:  "https://go.googlesource.com/",
		},
		{
			base: "https://go.googlesource.com/a%2fb",
			elem: []string{"c"},
			out:  "https://go.googlesource.com/a%2fb/c",
		},
		{
			base: "https://go.googlesource.com/a%2fb",
			elem: []string{"c%2fd"},
			out:  "https://go.googlesource.com/a%2fb/c%2fd",
		},
		{
			base: "https://go.googlesource.com/a/b",
			elem: []string{"/go"},
			out:  "https://go.googlesource.com/a/b/go",
		},
		{
			base: "https://go.googlesource.com/",
			elem: []string{"100%"},
		},
		{
			base: "/",
			elem: nil,
			out:  "/",
		},
		{
			base: "a",
			elem: nil,
			out:  "a",
		},
		{
			base: "a",
			elem: []string{"b"},
			out:  "a/b",
		},
		{
			base: "a",
			elem: []string{"../b"},
			out:  "b",
		},
		{
			base: "a",
			elem: []string{"../../b"},
			out:  "b",
		},
		{
			base: "",
			elem: []string{"a"},
			out:  "a",
		},
		{
			base: "",
			elem: []string{"../a"},
			out:  "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.base, func(t *testing.T) {
			out, err := JoinPath(tt.base, tt.elem...)
			if tt.out == "" {
				assert.Error(t, err, "JoinPath(%q, %q) should error", tt.base, tt.elem)
			} else {
				assert.NoError(t, err, "JoinPath(%q, %q) should not error", tt.base, tt.elem)
				assert.Equal(t, tt.out, out, "JoinPath(%q, %q) mismatch", tt.base, tt.elem)
			}

			u, err := Parse(tt.base)
			if err != nil {
				if tt.out != "" {
					t.Errorf("Parse(%q) = %v", tt.base, err)
				}
				return
			}
			if tt.out == "" {
				// URL.JoinPath doesn't return an error, so leave it unchanged
				tt.out = tt.base
			}
			out = URL_String(URL_JoinPath(u, tt.elem...))
			assert.Equal(t, tt.out, out, "Parse(%q).JoinPath(%q) mismatch", tt.base, tt.elem)
		})
	}
}

func TestClientPlaceholderInHost(t *testing.T) { //NOSONAR
	tests := []struct {
		name      string
		rawURL    string
		wantHost  string
		wantPath  string
		wantQuery string
	}{
		{
			name:     "placeholder as full host",
			rawURL:   "http://{client}/",
			wantHost: "{client}",
			wantPath: "/",
		},
		{
			name:     "placeholder as subdomain",
			rawURL:   "http://{client}.example.com/path",
			wantHost: "{client}.example.com",
			wantPath: "/path",
		},
		{
			name:     "placeholder in the middle of host",
			rawURL:   "http://api.{client}.com/",
			wantHost: "api.{client}.com",
			wantPath: "/",
		},
		{
			name:     "placeholder as full host with port",
			rawURL:   "http://{client}:8080/",
			wantHost: "{client}:8080",
			wantPath: "/",
		},
		{
			name:     "placeholder as subdomain with port",
			rawURL:   "http://{client}.example.com:8080/path",
			wantHost: "{client}.example.com:8080",
			wantPath: "/path",
		},
		{
			name:      "placeholder with query",
			rawURL:    "https://{client}.example.com/api?version=2",
			wantHost:  "{client}.example.com",
			wantPath:  "/api",
			wantQuery: "version=2",
		},
		{
			name:     "https scheme with placeholder",
			rawURL:   "https://{client}.example.com/",
			wantHost: "{client}.example.com",
			wantPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := Parse(tt.rawURL)
			require.NoError(t, err, "Parse(%q) returned unexpected error", tt.rawURL)
			assert.Equal(t, tt.wantHost, u.Host)
			assert.Equal(t, tt.wantPath, u.Path)
			assert.Equal(t, tt.wantQuery, u.RawQuery)
			got := URL_String(u)
			assert.Equal(t, tt.rawURL, got, "URL_String(Parse(%q)) should round-trip", tt.rawURL)
		})
	}
}

func TestInvalidClientPlaceholder(t *testing.T) {
	t.Run("malformed placeholder in host", func(t *testing.T) {
		tests := []struct {
			name   string
			rawURL string
		}{
			{"empty placeholder", "http://{}/"},
			{"double opening brace", "http://{{demo}.example.com/"},
			{"unclosed placeholder", "http://{demo.example.com/"},
			{"unmatched closing brace", "http://demo}.example.com/"},
			{"space inside placeholder", "http://{ client}.example.com/"},
			{"extra closing brace after placeholder", "http://{client}}.example.com/"},
			{"empty placeholder with port", "http://{}:8080/"},
			{"double opening brace with port", "http://{{demo}:8080/"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := Parse(tt.rawURL)
				assert.Error(t, err, "Parse(%q) should error for malformed placeholder", tt.rawURL)
			})
		}
	})

	t.Run("placeholder in query", func(t *testing.T) {
		tests := []struct {
			name   string
			rawURL string
		}{
			{"placeholder in query key", "http://example.com/?{client}=1"},
			{"placeholder in query value", "http://example.com/?x={client}"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := Parse(tt.rawURL)
				assert.Error(t, err, "Parse(%q) should error — placeholder is not allowed in query", tt.rawURL)
			})
		}
	})

	t.Run("placeholder in path is allowed", func(t *testing.T) {
		tests := []struct {
			name   string
			rawURL string
		}{
			{"placeholder in path segment", "http://example.com/{client}/path"},
			{"placeholder at end of path", "http://example.com/path/{client}"},
			{"placeholder as entire path", "http://example.com/{client}"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := Parse(tt.rawURL)
				assert.NoError(t, err, "Parse(%q) should allow placeholder in path", tt.rawURL)
			})
		}
	})
}
