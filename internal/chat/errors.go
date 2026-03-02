package chat

import "errors"

var (
	ErrSameUser = errors.New("chat: cannot create conversation with yourself")
	ErrNotFound = errors.New("chat: not found")
)
