// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"strings"
)

// replaceParameters returns a copy of the string s with all its parameters replaced
func replaceParameters(s string, params map[string]string) string {
	tmp := s
	for key, value := range params {
		tmp = strings.ReplaceAll(tmp, fmt.Sprint("$", key), value)
	}
	return tmp
}
