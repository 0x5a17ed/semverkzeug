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

	"github.com/0x5a17ed/semverkzeug/pkg/gitrepo"
)

type ContextKey int

const (
	GitRepositoryKey ContextKey = iota
	ScopeKey
)

func WithGitContext(ctx context.Context, repo *gitrepo.Context) context.Context {
	return context.WithValue(ctx, GitRepositoryKey, repo)
}

func GetGitContext(ctx context.Context) (repo *gitrepo.Context, ok bool) {
	repo, ok = ctx.Value(GitRepositoryKey).(*gitrepo.Context)
	return
}

func WithScope(ctx context.Context, scope gitrepo.Scope) context.Context {
	return context.WithValue(ctx, ScopeKey, scope)
}

func GetScope(ctx context.Context) (scope gitrepo.Scope, ok bool) {
	scope, ok = ctx.Value(ScopeKey).(gitrepo.Scope)
	return
}
