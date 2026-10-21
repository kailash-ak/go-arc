package main

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrorMsg = errors.New("age cannot be negative")

func ParseAge(age string) (int, error) {
	res, err := strconv.Atoi(age)
	if err != nil {
		wrappedErr := fmt.Errorf("parse age %q: %w", age, err)
		return 0, wrappedErr
	}
	if res < 0 {
		return 0, ErrorMsg
	}
	return res, nil
}
