package validators

import (
	v "github.com/gobuffalo/validate"
	"testing"
)

func TestHostValidator_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		construct    func() *HostValidator
		expectErrors bool
	}{
		{"Valid host", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "myexamplehost",
			}
		}, false},
		{"Empty host", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "",
			}
		}, true},
		{"Large host", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: createLongString(),
			}
		}, true},
		{"Host with port", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "host:8000",
			}
		}, true},
		{"Host with scheme", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "http://myhost",
			}
		}, true},
		{"Host with user", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "user@myhost",
			}
		}, true},
		{"Host with path", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "myhost/path",
			}
		}, true},
		{"Host with query", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "myhost?query=value",
			}
		}, true},
		{"Host with fragment", func() *HostValidator {
			return &HostValidator{
				Field: "TestField",
				Value: "myhost#fragment",
			}
		}, true},
		// More test cases for `Opaque`, `RawPath`, `RawFragment`, `EscapedPath`, `RequestURI`, `IsAbs` and others
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vErrors := v.NewErrors()
			test.construct().IsValid(vErrors)
			hasErrors := vErrors.HasAny()
			if test.expectErrors != hasErrors {
				t.Errorf("unexpected error status: got %v, want %v", hasErrors, test.expectErrors)
			}
		})
	}
}

// Auxiliary function that creates a long string
func createLongString() string {
	var s string
	for i := 0; i <= 255; i++ {
		s += "a"
	}
	return s
}
