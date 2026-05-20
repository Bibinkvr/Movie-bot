package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Int64s parses the environment variable as a slice of int64, ignoring any errors.
func Int64s(name string) (l []int64) {
	s := os.Getenv(name)
	if s == "" {
		return
	}

	for _, str := range strings.Fields(strings.ReplaceAll(s, ",", " ")) {
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			fmt.Println(err)
			continue
		}

		l = append(l, i)
	}

	return
}

// Strings parses the environment variable as a slice of string, ignoring any errors.
func Strings(name string) []string {
	s := os.Getenv(name)
	if s == "" {
		return nil
	}

	return strings.Fields(strings.ReplaceAll(s, ",", " "))
}

// Int64 parses the environment variable as an int64.
func Int64(name string) int64 {
	s := os.Getenv(name)
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// Int parses the environment variable as an int.
func Int(name string) int {
	s := os.Getenv(name)
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}
