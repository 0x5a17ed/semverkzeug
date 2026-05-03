/*
 * Copyright(C) 2026 the semverkzeug developers
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

package main

import (
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/0x5a17ed/semverkzeug/pkg/version"
)

// versionFlag, if given, prints the embedded program version and exits.
type versionFlag bool

func (versionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (versionFlag) IsBool() bool                         { return true }

// BeforeReset is called by kong as soon as the flag is encountered, so
// the version is printed before any command parsing side effects run.
func (versionFlag) BeforeReset(app *kong.Kong) error {
	_, _ = fmt.Fprintln(app.Stdout, version.Version)
	app.Exit(0)
	return nil
}

// cli is the top-level kong CLI grammar.
type cli struct {
	Repo string `short:"C" name:"repo" placeholder:"PATH" help:"git repository path (default is $PWD)"`

	Version versionFlag `name:"version" help:"Print version information and quit"`

	Describe describeCmd `cmd:"" help:"Print current version string"`
	Bump     bumpCmd     `cmd:"" help:"Bumps the current version and creates a new tag"`
}
