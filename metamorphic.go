// Copyright 2023 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package metamorphic provides facilities for running metamorphic,
// property-based testing. By running logically equivalent operations with
// different conditions, metamorphic tests can identify bugs without requiring
// an oracle.
package metamorphic

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
)

// Generate generates a sequence of n items, calling fn to produce
// each item. It's intended to be used with a function returned by
// (Weighted).Random or (Weighted).RandomDeck.
func Generate[I any](n int, fn func() I) []I {
	items := make([]I, n)
	for i := 0; i < n; i++ {
		items[i] = fn()
	}
	return items
}

// NewLogger constructs a new logger for running randomized tests.
func NewLogger(t testing.TB) *Logger {
	l := &Logger{t: t}
	// TODO(jackson): Support teeing to additional sink(s), eg, a file.
	l.w = &l.history
	l.wIndent = newlineIndentingWriter{Writer: l.w, indent: []byte("  ")}
	return l
}

// Step runs the provided operation against the provided state.
func Step[S any](l *Logger, s S, op Op[S]) {
	// Ensure panics result in printing the history.
	defer func() {
		if r := recover(); r != nil {
			l.Fatal(r)
		}
	}()

	// Set Logger's per-Op context.
	l.logged = false
	l.op = op
	fmt.Fprintf(l, "op %6d: %s = ", l.opNumber, l.op)
	op.Run(l, s)
	if !l.logged {
		fmt.Fprint(l, "-")
	}
	fmt.Fprintln(l)
	l.opNumber++
}

// Run runs the provided operations, using the provided initial state.
func Run[S any](t testing.TB, initial S, ops []Op[S]) {
	l := NewLogger(t)
	s := initial
	for i := 0; i < len(ops); i++ {
		Step[S](l, s, ops[i])
	}
	fmt.Fprintln(l, "done")
	if t.Failed() {
		t.Logf("History:\n\n%s", l.history.String())
	}
}

// RunInTandem takes n initial states and runs the provided set of operations
// against each incrementally. It fails the test as soon as any of the logs
// diverge.
func RunInTandem[S any](t testing.TB, initial []S, ops []Op[S]) []*Logger {
	logs := make([]*Logger, len(initial))
	logOpOffsets := make([][]int, len(initial))
	for i := 0; i < len(logs); i++ {
		logs[i] = &Logger{t: t}
		logs[i].w = &logs[i].history
		logs[i].wIndent = newlineIndentingWriter{Writer: logs[i].w, indent: []byte("  ")}
		logOpOffsets[i] = make([]int, len(ops))
	}

	s := initial
	for i := 0; i < len(ops); i++ {
		for j, l := range logs {
			func() {
				// Ensure panics result in printing the history.
				defer func() {
					if r := recover(); r != nil {
						l.Fatal(r)
					}
				}()

				// Set Logger's per-Op context.
				l.opNumber = i
				l.op = ops[i]
				l.logged = false
				logOpOffsets[j][i] = l.history.Len()
				fmt.Fprintf(l, "op %6d: %s = ", l.opNumber, l.op)
				ops[i].Run(l, s[j])

				if !l.logged {
					fmt.Fprint(l, "-")
				}
				fmt.Fprintln(l)
			}()
			if t.Failed() {
				t.Logf("Aborting; History:\n\n%s", l.history.String())
			}
			if j > 0 {
				if err := compareOpResults(logs[0], logOpOffsets[0][i], logs[j], logOpOffsets[j][i]); err != nil {
					t.Errorf("state %d and %d diverged at op %d:\n%s", 0, j, i, err)
				}
			}
		}
	}
	return logs
}

// Op represents a single operation within a metamorphic test.
type Op[S any] interface {
	fmt.Stringer

	// Run runs the operation, logging its outcome to Logger.
	Run(*Logger, S)
}

// Logger logs test operation's outputs and errors, maintaining a cumulative
// history of the test. Logger may be used analogously to the standard library's
// testing.TB.
type Logger struct {
	w        io.Writer
	wIndent  newlineIndentingWriter
	t        testing.TB
	history  bytes.Buffer
	lastByte byte

	// op context; updated before each operation is run
	opNumber int
	op       fmt.Stringer
	logged   bool
}

// Assert that *Logger implements require.TestingT.
var _ require.TestingT = (*Logger)(nil)

// Commentf writes a comment to the log file. Commentf always appends a newline
// after the comment. Commentf may prepend a newline before the message if there
// isn't already one.
func (l *Logger) Commentf(format string, args ...any) {
	if l.lastByte != '\n' {
		fmt.Fprintln(&l.wIndent)
	}
	l.Logf("// "+format+"\n", args...)
}

// Error fails the test run, logging the provided error.
func (l *Logger) Error(err error) {
	l.Log("error: ", err)
	l.t.Error(err)
}

// Errorf fails the test run, logging the provided message.
func (l *Logger) Errorf(format string, args ...any) {
	l.Logf("error: "+format, args...)
	l.t.Errorf(format, args...)
}

// FailNow marks the function as having failed and stops its execution by
// calling runtime.Goexit. FailNow is implemented by calling through to the
// underlying *testing.T's FailNow.
func (l *Logger) FailNow() {
	l.t.Logf("History:\n\n%s", l.history.String())
	l.t.FailNow()
}

// Fatal is equivalent to Log followed by FailNow.
func (l *Logger) Fatal(args ...any) {
	l.Errorf("%s", fmt.Sprint(args...))
	l.FailNow()
}

// Log formats its arguments using default formatting, analogous to Print, and
// records the text in the test's recorded history.
func (l *Logger) Log(args ...any) {
	l.logged = true
	fmt.Fprint(&l.wIndent, args...)
}

// Logf formats its arguments according to the format, analogous to Printf, and
// records the text in the test's recorded history.
func (l *Logger) Logf(format string, args ...interface{}) {
	l.logged = true
	fmt.Fprintf(&l.wIndent, format, args...)
}

// Write implements io.Writer.
func (l *Logger) Write(b []byte) (int, error) {
	n, err := l.w.Write(b)
	if n > 0 {
		l.lastByte = b[n-1]
	}
	return n, err
}

// History returns the history accumulated by the Logger.
func (l *Logger) History() string {
	return l.history.String()
}

func compareOpResults(a *Logger, offA int, b *Logger, offB int) error {
	aResult := a.history.Bytes()[offA:]
	bResult := b.history.Bytes()[offB:]
	if !bytes.Equal(aResult, bResult) {
		return errors.Newf("divergence:\n%s\n%s\n", aResult, bResult)
	}
	return nil
}

// newlineIndentingWriter wraps a Writer. Whenever a '\n' is written, the
// newlineIndentingWriter writes the '\n' and the configured `indent` byte slice
// to the writer. All other bytes written are written to the underlying Writer
// verbatim.
type newlineIndentingWriter struct {
	io.Writer
	indent []byte
}

func (w *newlineIndentingWriter) Write(b []byte) (n int, err error) {
	for len(b) > 0 {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			n2, err := w.Writer.Write(b[:i+1])
			n += n2
			if err != nil {
				return n, err
			}
			b = b[i+1:]
			n2, err = w.Writer.Write(w.indent)
			n += n2
			if err != nil {
				return n, err
			}
			continue
		}

		n2, err := w.Writer.Write(b)
		n += n2
		return n, err
	}
	return n, err
}
