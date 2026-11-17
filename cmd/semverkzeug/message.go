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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// stdinPath is the --message-file argument that selects standard input.
const stdinPath = "-"

var errEmptyMessage = errors.New("message is empty")

// nonEmptyMessage turns read content into a tag message.  It rejects
// content without visible characters, because git would refuse such a
// message or the tag would carry no information.
func nonEmptyMessage(content []byte) (string, error) {
	message := string(content)
	if strings.TrimSpace(message) == "" {
		return "", errEmptyMessage
	}
	return message, nil
}

// messageVariant is one of the places a tag message can come from.
// Each variant knows how to read its own message, so callers never
// branch on where the message comes from.
type messageVariant interface {
	read(stdin io.Reader) (string, error)
}

// MessageGeneratedStub selects the message that the bumper generates.
type MessageGeneratedStub struct{}

// read returns an empty message, which tells the bumper to generate one.
func (MessageGeneratedStub) read(io.Reader) (string, error) {
	return "", nil
}

// MessageFromStdin reads the tag message from standard input.
type MessageFromStdin struct{}

func (MessageFromStdin) read(stdin io.Reader) (string, error) {
	content, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read standard input: %w", err)
	}
	return nonEmptyMessage(content)
}

// MessageFromFile reads the tag message from the file at path.
type MessageFromFile struct {
	path string
}

func (m MessageFromFile) read(io.Reader) (string, error) {
	content, err := os.ReadFile(m.path)
	if err != nil {
		return "", err
	}
	return nonEmptyMessage(content)
}

// MessageSource holds the place the tag message comes from.  It is
// one of [MessageGeneratedStub], [MessageFromStdin] or
// [MessageFromFile].
//
// The zero value selects the generated message, so a --message-file
// flag that is not given needs no default.
type MessageSource struct {
	variant messageVariant
}

// ParseMessageSource parses a --message-file argument.  The argument
// "-" selects standard input and any other argument names a file.
//
// An empty argument names no file and is rejected.  Leave the flag out
// to select the generated message.
func ParseMessageSource(raw string) (MessageSource, error) {
	if raw == "" {
		return MessageSource{}, errors.New("empty path")
	}
	if raw == stdinPath {
		return MessageSource{variant: MessageFromStdin{}}, nil
	}
	return MessageSource{variant: MessageFromFile{path: raw}}, nil
}

// UnmarshalText parses raw via [ParseMessageSource] and writes the
// result into s.  It satisfies [encoding.TextUnmarshaler], which kong
// uses to decode the flag argument straight into the strong type.
func (s *MessageSource) UnmarshalText(raw []byte) error {
	parsed, err := ParseMessageSource(string(raw))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// held returns the variant that s holds, and maps the zero value to
// [MessageGeneratedStub].
func (s MessageSource) held() messageVariant {
	if s.variant == nil {
		return MessageGeneratedStub{}
	}
	return s.variant
}

// Read returns the tag message.  It reads stdin only when s selects
// standard input.  An empty message tells the bumper to generate one.
func (s MessageSource) Read(stdin io.Reader) (string, error) {
	message, err := s.held().read(stdin)
	if err != nil {
		return "", fmt.Errorf("read tag message: %w", err)
	}
	return message, nil
}
