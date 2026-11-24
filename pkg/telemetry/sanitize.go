package telemetry

import (
	"net/url"
	"regexp"
	"strings"
)

// SanitizeURL returns raw without userinfo, query and fragment, and with the
// path segment after "secrets" redacted. raw may be a URL, an scp-like Git
// address or a bare reference. When raw doesn't parse cleanly, everything up
// to its last '@' is dropped.
func SanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.Scheme != "" && u.Host != "" && !missedUserinfo(u, raw) {
		u.User = nil
		u.RawQuery = ""
		u.ForceQuery = false
		u.Fragment = ""
		u.RawFragment = ""
		return secretPathPattern.ReplaceAllString(u.String(), "${1}{redacted}")
	}
	s := raw
	if i := strings.Index(s, "://"); i >= 0 {
		if at := userinfoEnd(s[i+3:]); at >= 0 {
			s = s[:i+3] + s[i+3+at+1:]
		}
	} else if at := strings.LastIndex(s, "@"); at >= 0 && looksLikeUserinfo(s[:at]) {
		s = s[at+1:]
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return secretPathPattern.ReplaceAllString(s, "${1}{redacted}")
}

// secretPathPattern matches the path segment after "secrets", such as a
// Secret's name in a Kubernetes API path.
var secretPathPattern = regexp.MustCompile(`(/secrets/)[^/]+`)

// missedUserinfo returns whether url.Parse read userinfo as host and port, as
// it does for user:12/ss@host.
func missedUserinfo(u *url.URL, raw string) bool {
	if u.User != nil || u.Port() == "" || !strings.Contains(raw, "@") {
		return false
	}
	if u.Path == "" {
		return true
	}
	first, _, _ := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
	return strings.Index(first, "@") > 0
}

// userinfoEnd returns the index of the '@' ending the userinfo of rest, the
// part after "://", or -1: the last '@' before the first '/' or, without one,
// the last '@' of all.
func userinfoEnd(rest string) int {
	authority, _, _ := strings.Cut(rest, "/")
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		return at
	}
	return strings.LastIndex(rest, "@")
}

// looksLikeUserinfo returns whether s, a scheme-less reference up to its last
// '@', is userinfo: no path, or a ':' before one.
func looksLikeUserinfo(s string) bool {
	slash := strings.Index(s, "/")
	if slash < 0 {
		return true
	}
	colon := strings.Index(s, ":")
	return colon >= 0 && colon < slash
}

// urlPattern matches URLs with a scheme in free text. A URL ends at
// whitespace, a quote (\x60 is a backtick) or an angle bracket, except in a
// password. It over-matches "https://host:80 for a@b" as "https://b".
var urlPattern = regexp.MustCompile(
	schemePattern.String() +
		`(?:[^\s:/?#@"'\x60<>]+:[^\r\n/?#@"'\x60<>]*@)?` +
		`[^\s"'\x60<>]+`,
)

// schemePattern matches the start of a URL with a scheme.
var schemePattern = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://`)

// RedactURLs returns s with every URL in it passed through SanitizeURL. A URL
// within another, such as after a comma or a proxy's address, is passed
// through on its own.
func RedactURLs(s string) string {
	if !strings.Contains(s, "://") {
		return s
	}
	return urlPattern.ReplaceAllStringFunc(s, func(u string) string {
		if loc := schemePattern.FindAllStringIndex(u, 2); len(loc) == 2 {
			return SanitizeURL(u[:loc[1][0]]) + RedactURLs(u[loc[1][0]:])
		}
		return SanitizeURL(u)
	})
}

// secretNamePattern matches a quoted Secret name, such as secrets "x".
var secretNamePattern = regexp.MustCompile(`\b([Ss]ecrets? )"[^"\s]+"`)

// redactSecretNames returns s with every quoted Secret name in it redacted.
func redactSecretNames(s string) string {
	if !strings.Contains(s, "ecret") {
		return s
	}
	return secretNamePattern.ReplaceAllString(s, `${1}"{redacted}"`)
}
