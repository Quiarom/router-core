package main

import "time"

// nowRFC3339 returns the current time as an RFC3339 string.
// Centralized so tests can stub it if needed.
var nowRFC3339 = func() string {
	return time.Now().UTC().Format(time.RFC3339)
}
