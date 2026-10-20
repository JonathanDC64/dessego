package game

import (
	"net/http"
	"strings"
)

// clientOptionsHeader carries options a client sets for itself, as key=value
// pairs separated by ";" (the Demon's Souls PC port sends it):
//
//	group=<1-64 lowercase hex digits>
//	    The matchmaking group: a hash of a password its players share. Signs,
//	    invasions and the Old Monk only match within a group; messages,
//	    bloodstains, ghosts and replays stay shared by everyone. Without it a
//	    client is in the public group "".
//
// Unknown keys and malformed values are ignored.
const clientOptionsHeader = "X-DeS-Options"

// clientOption returns the value of one key in the request's options header,
// or "" when it is not there.
func clientOption(r *http.Request, key string) string {
	for _, kv := range strings.Split(r.Header.Get(clientOptionsHeader), ";") {
		if i := strings.Index(kv, "="); i >= 0 && strings.TrimSpace(kv[:i]) == key {
			return strings.TrimSpace(kv[i+1:])
		}
	}
	return ""
}

// matchGroup returns the request's matchmaking group, "" (public) when it has
// none or it is malformed.
func matchGroup(r *http.Request) string {
	g := clientOption(r, "group")
	if len(g) == 0 || len(g) > 64 {
		return ""
	}
	for _, c := range g {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ""
		}
	}
	return g
}
