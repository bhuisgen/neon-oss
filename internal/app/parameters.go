// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"regexp"
	"strings"
)

var (
	regexpParameters = regexp.MustCompile(`\:(.[^\:/]+)`)
)

// FindParameters finds the parameters in a string
func FindParameters(s string) [][]string {
	namedParams := regexpParameters.FindAllStringSubmatch(s, -1)

	return namedParams
}

// ReplaceParameters replaces the parameters in a string by the given values
func ReplaceParameters(s string, paramsKeys []string, paramsValues []string) string {
	tmp := s
	keyParams := regexpParameters.FindAllString(s, -1)
	for _, keyParam := range keyParams {
		for index, param := range paramsKeys {
			if param == keyParam {
				tmp = strings.Replace(tmp, keyParam, paramsValues[index], -1)
			}
		}
	}

	return tmp
}
