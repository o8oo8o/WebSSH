// Package pq is a pure Go Postgres driver for the database/sql package.

//go:build js || android || hurd || zos
// +build js android hurd zos

package pgsql

func userCurrent() (string, error) {
	return "", ErrCouldNotDetectUsername
}
