// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package normalize

import (
	"errors"
	"slices"
	"strconv"
)

// mapKeySegment stands in for map keys in error paths so that keys, which may
// hold user data, never reach error messages.
const mapKeySegment = "{key}"

// segment is one step of a traversal path: a field name, mapKeySegment, or,
// when name is empty, a slice or array index.
type segment struct {
	name  string
	index int
}

// pathError carries a traversal failure up to the entry point. Each level it
// passes adds its segment, so successful calls never build a path.
type pathError struct {
	// segments lists the path innermost first.
	segments []segment
	// build creates the final error once the path is known.
	build func(path string) error
}

func (e *pathError) Error() string { return e.resolve().Error() }

func (e *pathError) resolve() error { return e.build(formatPath(e.segments)) }

func failAt(build func(path string) error) error { return &pathError{build: build} }

// within adds s to the path of err.
func within(err error, s segment) error {
	if pe, ok := errors.AsType[*pathError](err); ok {
		pe.segments = append(pe.segments, s)
	}
	return err
}

// resolvePath turns a pathError into the error it describes.
func resolvePath(err error) error {
	if pe, ok := errors.AsType[*pathError](err); ok {
		return pe.resolve()
	}
	return err
}

// formatPath renders innermost-first segments as Field.Items[1].{key}.Name.
func formatPath(segments []segment) string {
	var b []byte
	for _, s := range slices.Backward(segments) {
		if s.name == "" {
			b = append(b, '[')
			b = strconv.AppendInt(b, int64(s.index), 10)
			b = append(b, ']')
			continue
		}
		if len(b) > 0 {
			b = append(b, '.')
		}
		b = append(b, s.name...)
	}
	return string(b)
}
