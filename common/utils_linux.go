package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func FromSysfsNumber(dst *int64, base int) func(string) error {
	return func(name string) error {
		*dst = -1

		file, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("unable to open '%s' for reading: %w", name, err)
		}
		defer file.Close()
		str, err := bufio.NewReader(file).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("unable to read '%s': %w", name, err)
		}
		if str = strings.TrimSuffix(str, "\n"); len(str) == 0 {
			return fmt.Errorf("unable to get id from '%s'", name)
		}
		id, err := strconv.ParseInt(str, base, 64)
		if err != nil {
			return fmt.Errorf("unable to parse out id from '%s'; %w", name, err)
		}
		*dst = id
		return nil
	}
}

func FromSysfsString(dst *string) func(string) error {
	return func(name string) error {
		*dst = ""

		file, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("unable to open '%s' for reading: %w", name, err)
		}
		defer file.Close()
		str, err := bufio.NewReader(file).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("unable to read '%s': %w", name, err)
		}
		*dst = strings.TrimSuffix(str, "\n")
		return nil
	}
}
