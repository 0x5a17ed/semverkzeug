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
	"os"

	"github.com/alecthomas/kong"

	"github.com/0x5a17ed/semverkzeug/internal/uiprint"
)

func main() {
	var grammar cli
	kctx := kong.Parse(&grammar,
		kong.Name("semverkzeug"),
		kong.Description("versioning tool for git repositories"),
		kong.UsageOnError(),
	)

	// Register lazy singleton providers so each command's Run() can
	// just declare the dependencies it needs (gitrepo.Context, HEAD
	// reference, ...) and kong wires them up exactly once per call.
	for _, err := range []error{
		kctx.BindSingletonProvider(provideRepo),
		kctx.BindSingletonProvider(provideHead),
	} {
		kctx.FatalIfErrorf(err)
	}

	if err := kctx.Run(); err != nil {
		uiprint.Error("%s", err.Error())
		os.Exit(1)
	}
}
