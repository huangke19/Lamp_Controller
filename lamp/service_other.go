//go:build !darwin

package main

import "fmt"

func runServiceCommand(args []string) error {
	return fmt.Errorf("service command is only supported on macOS")
}
