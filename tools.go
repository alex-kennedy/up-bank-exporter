//go:build tools

// Package tools ensures tools are installed and managed by the go.mod file.
package tools

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/oapi-codegen/runtime"
)
