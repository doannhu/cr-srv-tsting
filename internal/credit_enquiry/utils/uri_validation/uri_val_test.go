package uri_validation_test

import (
	"testing"

	uri_errors "go-loan-service-v3/internal/credit_enquiry/errors/uri_validation_errors"
	uri_validation "go-loan-service-v3/internal/credit_enquiry/utils/uri_validation"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		s    string
		ok   bool
		code string
	}{
		// ————— valid —————
		{"http://example.com", true, ""},
		{"https://user@example.com:8080/path?query#frag", true, ""},
		{"mailto:foo@example.com", true, ""},
		{"urn:example:animal:ferret:nose", true, ""},
		{"scheme+ext.-1://host", true, ""},

		// ————— invalid scheme —————
		{"1http://example.com", false, uri_errors.ErrInvalidScheme},
		{"://missing.scheme", false, uri_errors.ErrInvalidScheme},

		// ————— invalid host —————
		// {"http://256.0.0.1", false, uri_errors.ErrInvalidHost},
		{"http://[:::1]", false, uri_errors.ErrInvalidHost},

		// ————— invalid port —————
		{"http://example.com:99999", false, uri_errors.ErrInvalidPort},
		{"http://example.com:port", false, uri_errors.ErrInvalidPort},

		// ————— bad percent —————
		{"http://example.com/%zz", false, uri_errors.ErrInvalidPath},

		// ————— invalid query char —————
		{"http://example.com?bad|char", false, uri_errors.ErrInvalidQuery},
	}

	for _, c := range cases {
		err := uri_validation.Validate(c.s)
		if c.ok {
			if err != nil {
				t.Errorf("%q expected ok, got %v", c.s, err)
			}
		} else {
			if err == nil {
				t.Errorf("%q expected error %v, got nil", c.s, c.code)
				continue
			}
			uerr, ok := err.(*uri_errors.Error)
			if !ok {
				t.Errorf("%q returned non‑uri error %v", c.s, err)
				continue
			}
			if uerr.Code != c.code {
				t.Errorf("%q expected code %v, got %v", c.s, c.code, uerr.Code)
			}
		}
	}
}

// FuzzValidate ensures Validate never panics.
func FuzzValidate(f *testing.F) {
	seeds := []string{
		"http://example.com",
		"https://[::1]",
		"mailto:a@b",
		"bad_scheme://",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = uri_validation.Validate(s)
	})
}
