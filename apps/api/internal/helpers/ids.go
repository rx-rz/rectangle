package helpers

import "github.com/nrednav/cuid2"

func NewUserID() string {
	id := cuid2.Generate()
	return "user_" + id
}

func NewSessionID() string {
	id := cuid2.Generate()
	return "session_" + id
}
