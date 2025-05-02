package uri_validation

import (
	"net"
	"regexp"
	"strconv"
	"strings"

	uri_validation_errors "go-loan-service-v3/internal/credit_enquiry/errors/uri_validation_errors"

	"golang.org/x/net/idna"
)

/* -------------------------------------------------------------------------
 * Pre‑compiled ABNF fragments (regex)
 * -------------------------------------------------------------------------*/

var (
	reScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+\-.]*$`) // §3.1
	reIPv4   = regexp.MustCompile(`^(?:25[0-5]|2[0-4]\d|[01]?\d?\d)(?:\.(?:25[0-5]|2[0-4]\d|[01]?\d?\d)){3}$`)
	// Simplified IPv6 literal (covers all legal forms, rejects most bad ones)
	reIPv6Lit  = regexp.MustCompile(`^\[[0-9A-Fa-f:.]+\]$`) // further checked by net.ParseIP after trim
	rePort     = regexp.MustCompile(`^[0-9]*$`)
	reRegName  = regexp.MustCompile(`^(?:%[0-9A-Fa-f]{2}|[A-Za-z0-9\-._~!$&'()*+,;=])+$`)
	rePathChar = regexp.MustCompile(`^(?:%[0-9A-Fa-f]{2}|[A-Za-z0-9\-._~!$&'()*+,;=:@/])*$`)
	reQueryFrg = regexp.MustCompile(`^(?:%[0-9A-Fa-f]{2}|[A-Za-z0-9\-._~!$&'()*+,;=:@/?])*$`)
)

/* -------------------------------------------------------------------------
 * Public API
 * -------------------------------------------------------------------------*/

// Validate checks s for full compliance with RFC 3986.  It returns nil when
// syntactically valid, otherwise an *Error describing the first problem found.
func Validate(s string) error {
	if len(s) == 0 {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidScheme, Field: "scheme", Msg: "empty string"}
	}

	// Split into main components: <scheme> ':' <rest>
	schemeEnd := strings.IndexByte(s, ':')
	if schemeEnd == -1 {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidScheme, Field: "scheme", Msg: "missing ':'"}
	}
	scheme := s[:schemeEnd]
	rest := s[schemeEnd+1:]

	if !reScheme.MatchString(scheme) {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidScheme, Field: "scheme", Msg: "invalid characters"}
	}

	// Optional authority starts with "//" (§3)
	var authority string
	if strings.HasPrefix(rest, "//") {
		// authority is up to next '/' '?' or '#'
		aEnd := len(rest)
		for i, c := range rest[2:] {
			if c == '/' || c == '?' || c == '#' {
				aEnd = 2 + i
				break
			}
		}
		authority = rest[2:aEnd]
		rest = rest[aEnd:]
		if err := validateAuthority(authority); err != nil {
			return err
		}
	}

	// Split path, query, fragment
	path, query, frag := splitPathQueryFrag(rest)

	if err := validatePath(path); err != nil {
		return err
	}
	if err := validateQueryOrFrag(query, "query", uri_validation_errors.ErrInvalidQuery); err != nil {
		return err
	}
	if err := validateQueryOrFrag(frag, "fragment", uri_validation_errors.ErrInvalidFragment); err != nil {
		return err
	}
	return nil
}

/* -------------------------------------------------------------------------
 * Component validators
 * -------------------------------------------------------------------------*/

func validateAuthority(auth string) error {
	// userinfo@host:port
	var userinfo, hostport string
	if at := strings.LastIndex(auth, "@"); at != -1 {
		userinfo = auth[:at]
		hostport = auth[at+1:]
		if err := validateUserinfo(userinfo); err != nil {
			return err
		}
	} else {
		hostport = auth
	}

	host, port := splitHostPort(hostport)
	if err := validateHost(host); err != nil {
		return err
	}
	if port != "" {
		if !rePort.MatchString(port) {
			return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidPort, Field: "port", Msg: "non‑digit characters"}
		}
		if p, _ := strconv.Atoi(port); p > 65535 {
			return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidPort, Field: "port", Msg: "must be 0‑65535"}
		}
	}
	return nil
}

func validateUserinfo(ui string) error {
	if ui == "" {
		return nil
	}
	// userinfo allows pchar plus ':'
	if !rePathChar.MatchString(ui) {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidUserinfo, Field: "userinfo", Msg: "invalid character"}
	}
	return nil
}

func validateHost(h string) error {
	if h == "" {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidHost, Field: "host", Msg: "empty"}
	}
	switch {
	case reIPv6Lit.MatchString(h):
		// Strip brackets and rely on net.ParseIP for full validation.
		ip := strings.Trim(h, "[]")
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() != nil {
			return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidHost, Field: "host", Msg: "malformed IPv6 literal"}
		}
		return nil
	case reIPv4.MatchString(h):
		return nil
	case reRegName.MatchString(h):
		if _, err := idna.Lookup.ToASCII(h); err != nil {
			return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidHost, Field: "host", Msg: err.Error()}
		}
		return nil
	default:
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidHost, Field: "host", Msg: "invalid format"}
	}
}

func validatePath(p string) error {
	// Paths may be empty
	if p == "" {
		return nil
	}
	if !rePathChar.MatchString(p) {
		return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidPath, Field: "path", Msg: "invalid character"}
	}
	if err := validatePercents(p, "path"); err != nil {
		return err
	}
	return nil
}

func validateQueryOrFrag(s, field string, code string) error {
	if s == "" {
		return nil
	}
	if !reQueryFrg.MatchString(s) {
		return &uri_validation_errors.Error{Code: code, Field: field, Msg: "invalid character"}
	}
	if err := validatePercents(s, field); err != nil {
		return err
	}
	return nil
}

/* -------------------------------------------------------------------------
 * Helpers
 * -------------------------------------------------------------------------*/

// splitHostPort splits host[:port] where port is optional and may be empty.
func splitHostPort(s string) (host, port string) {
	if i := strings.LastIndex(s, ":"); i != -1 && !strings.Contains(s[i+1:], "]") && !strings.Contains(s[:i], "]") {
		// last ':' not inside IPv6 literal
		host, port = s[:i], s[i+1:]
	} else {
		host = s
	}
	return
}

// splitPathQueryFrag returns path, query, fragment from <path>[?query][#frag].
func splitPathQueryFrag(rest string) (path, query, frag string) {
	if rest == "" {
		return "", "", ""
	}
	q := strings.IndexByte(rest, '?')
	h := strings.IndexByte(rest, '#')

	switch {
	case q == -1 && h == -1:
		// only path
		path = rest
	case q != -1 && (h == -1 || q < h):
		// path?query[#frag]
		path = rest[:q]
		if h == -1 {
			query = rest[q+1:]
		} else {
			query = rest[q+1 : h]
			frag = rest[h+1:]
		}
	default:
		// path#frag (no query) OR '#'
		path = rest[:h]
		frag = rest[h+1:]
	}
	return
}

func validatePercents(s, field string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
				return &uri_validation_errors.Error{Code: uri_validation_errors.ErrInvalidPercent, Field: field, Msg: "bad percent‑encoding"}
			}
			i += 2
		}
	}
	return nil
}

func isHex(b byte) bool {
	return ('0' <= b && b <= '9') || ('A' <= b && b <= 'F') || ('a' <= b && b <= 'f')
}
