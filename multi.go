// © 2021 John Lenton. MIT licensed. from https://chipaca.com/gofomash
package main // import "chipaca.com/gofomash"

import (
	"strings"
)

type multi []string

func (m multi) String() string {
	return strings.Join([]string(m), ",")
}

func (m *multi) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func (m *multi) Get() interface{} {
	return *m
}
