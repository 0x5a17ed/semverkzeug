/*
 * Copyright(C) 2022 individual contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * <https://www.apache.org/licenses/LICENSE-2.0>
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied.  See the License for the specific
 * language governing permissions and limitations under the License.
 */

// Package uiprint emits pacman/makepkg-style progress chatter to
// stderr.  Output uses three prefix levels:
//
//	==>   top-level step
//	 ->   sub-step
//	   :: hint or aside
package uiprint

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

var (
	stepPrefix    = color.New(color.FgGreen, color.Bold)
	substepPrefix = color.New(color.FgBlue, color.Bold)
	hintPrefix    = color.New(color.FgYellow, color.Bold)
	errorPrefix   = color.New(color.FgRed, color.Bold)
	boldText      = color.New(color.Bold)

	// out is the destination for all messages.  Stderr keeps stdout
	// clean for callers that pipe the tool's data output.
	out io.Writer = os.Stderr
)

func init() {
	// fatih/color's auto-detection only checks stdout; we emit to
	// stderr, so re-evaluate against that descriptor.
	if f, ok := out.(*os.File); ok && !isatty.IsTerminal(f.Fd()) && !isatty.IsCygwinTerminal(f.Fd()) {
		color.NoColor = true
	}
}

// Step prints a top-level step line.
func Step(format string, args ...any) {
	_, _ = stepPrefix.Fprint(out, "==> ")
	_, _ = boldText.Fprintln(out, fmt.Sprintf(format, args...))
}

// Substep prints a sub-step line.
func Substep(format string, args ...any) {
	_, _ = substepPrefix.Fprint(out, " -> ")
	_, _ = fmt.Fprintln(out, fmt.Sprintf(format, args...))
}

// Hint prints a parenthetical aside.
func Hint(format string, args ...any) {
	_, _ = hintPrefix.Fprint(out, "   :: ")
	_, _ = fmt.Fprintln(out, fmt.Sprintf(format, args...))
}

// Error prints a pacman/makepkg-style error line.
func Error(format string, args ...any) {
	_, _ = errorPrefix.Fprint(out, "==> ERROR: ")
	_, _ = fmt.Fprintln(out, fmt.Sprintf(format, args...))
}
