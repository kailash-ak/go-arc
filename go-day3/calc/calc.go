package calc

import "errors"

var ErrorDivideByZero = errors.New("cannot divide by zero")

func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, ErrorDivideByZero
	}
	return a / b, nil
}
