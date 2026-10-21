package main

import (
	"errors"
	"fmt"
	"go-day3/calc"
	"strconv"
)

var ErrNotfound = errors.New("user not found")

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("can't divide with 0")
	}
	return a / b, nil
}

func getUser(id int) (string, error) {
	//var ErrNotfound = errors.New("user not found")
	if id == 0 {
		return "kailash", nil
	} else if id == 1 {
		return "mahesh", nil
	} else {
		return "", ErrNotfound
	}
}

func main() {
	s := "1a23"
	v, err := strconv.Atoi(s)
	fmt.Printf("type:%T", err)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(v)
	res, er := divide(10, 2)
	if er != nil {
		fmt.Println(er)

	}
	fmt.Println(res)
	id := 0
	//var ErrNotfound = errors.New("user not found")
	user, erro := getUser(id)
	if erro != nil {
		wrappedErr := fmt.Errorf("get user failed %w ", erro)
		fmt.Println(errors.Is(wrappedErr, ErrNotfound))

	} else {
		fmt.Printf("user with %d is found:%s", id, user)
	}
	runCustomErrorExercise()
	result, errorr := calc.Divide(10, 2)
	fmt.Println(result, errorr)
	if errorr != nil {
		fmt.Println(errorr)
		fmt.Println(errors.Is(errorr, calc.ErrorDivideByZero))
	}
	fmt.Println(result)
	age, aerr := ParseAge("abc")
	fmt.Println(age, aerr)

}
