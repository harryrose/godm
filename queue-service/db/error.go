package db

type ErrConflict struct{}
type ErrNotFound struct{}
type ErrInvalid struct{}

func (e ErrConflict) Error() string { return "conflict" }
func (e ErrNotFound) Error() string { return "not found" }
func (e ErrInvalid) Error() string  { return "invalid input" }
