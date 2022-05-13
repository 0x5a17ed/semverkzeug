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

package cli

import (
	"context"

	"github.com/go-git/go-git/v5"
)

type ContextKey int

const (
	GitRepositoryKey ContextKey = iota
)

func WithGitRepository(ctx context.Context, repo *git.Repository) context.Context {
	return context.WithValue(ctx, GitRepositoryKey, repo)
}

func GetGitRepository(ctx context.Context) (repo *git.Repository, ok bool) {
	repo, ok = ctx.Value(GitRepositoryKey).(*git.Repository)
	return
}
