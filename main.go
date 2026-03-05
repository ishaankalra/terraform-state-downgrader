// Copyright (c) 2024 Ishaan Kalra
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/ishaankalra/terraform-state-downgrade/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}