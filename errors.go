package gosmig

import "errors"

var (
	errDBVersionChangedUp = errors.New(
		"database version changed while applying migration up")
	errDBVersionChangedDown = errors.New(
		"database version changed while applying migration down")
)
