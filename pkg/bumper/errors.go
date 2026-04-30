package bumper

import (
	"errors"
)

var (
	ErrRepositoryIsEmpty = errors.New("repository is empty")
	ErrRepositoryIsDirty = errors.New("repository contains uncommitted changes")
)
