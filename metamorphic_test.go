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

package metamorphic

import (
	"fmt"
	"testing"
)

type mapOpKind int8

const (
	mapOpPut mapOpKind = iota
	mapOpDel
	mapOpGet
)

type mapOp struct {
	kind mapOpKind
	k    string
	v    string
}

func (o mapOp) String() string {
	switch o.kind {
	case mapOpPut:
		return fmt.Sprintf("Put(%q, %q)", o.k, o.v)
	case mapOpDel:
		return fmt.Sprintf("Del(%q)", o.k)
	case mapOpGet:
		return fmt.Sprintf("Get(%q)", o.k)
	default:
		return "unknown"
	}
}

func (o mapOp) Run(l *Logger, m map[string]string) {
	switch o.kind {
	case mapOpPut:
		m[o.k] = o.v
	case mapOpDel:
		delete(m, o.k)
	case mapOpGet:
		l.Logf("%q", m[o.k])
	}
}

func TestRunInTandem(t *testing.T) {
	initial := []map[string]string{
		make(map[string]string),
		make(map[string]string),
		make(map[string]string),
	}
	ops := []Op[map[string]string]{
		mapOp{kind: mapOpPut, k: "foo", v: "hello world"},
		mapOp{kind: mapOpDel, k: "bar"},
		mapOp{kind: mapOpGet, k: "bar"},
		mapOp{kind: mapOpGet, k: "foo"},
		mapOp{kind: mapOpPut, k: "bar", v: "bonjour monde"},
		mapOp{kind: mapOpGet, k: "bar"},
		mapOp{kind: mapOpDel, k: "bar"},
		mapOp{kind: mapOpGet, k: "bar"},
	}
	logs := RunInTandem(t, initial, ops)
	t.Logf("\n%s\n", logs[0].History())
}
