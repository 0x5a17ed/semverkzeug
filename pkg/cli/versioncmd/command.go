/*
 * Copyright(C) 2025 individual contributors
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

package versioncmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0x5a17ed/semverkzeug/pkg/cli"
	"github.com/0x5a17ed/semverkzeug/pkg/version"
)

func runE(ctx context.Context, cmd *cobra.Command, args []string) error {
	v, err := version.GetVersion()
	if err != nil {
		return err
	}

	fmt.Println(v)

	return nil
}

func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Print current version of the program itself",
		Run:   cli.RunCatchErr(runE),
	}

	return c
}
