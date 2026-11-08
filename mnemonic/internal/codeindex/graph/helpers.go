package graph

import "time"

// nowTimestamp returns the current timestamp in RFC3339 format.
func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
