package main

import (
	"errors"
	"fmt"
)

type UserNotFoundError struct {
	UserID int
}

func (e UserNotFoundError) Error() string {
	return fmt.Sprintf("user with %d id not found", e.UserID)
}

// TODO 1: Implement Error() string for UserNotFoundError.

func findUser(id int) (string, error) {
	if id == 1 {
		return "Kailash", nil
	} else {
		return "", UserNotFoundError{UserID: id}
	}

	// TODO 2: Return a UserNotFoundError containing the requested ID.
	//return "", nil
}

func runCustomErrorExercise() {

	name, err := findUser(90)

	// TODO 5: If extraction succeeds, print the missing user's ID.
	if err != nil {
		// TODO 3: Wrap err with the message "user lookup failed".
		var wrappedErr = fmt.Errorf("user lookup failed: %w", err)

		// TODO 4: Use errors.As() to extract UserNotFoundError.
		var usernotfounderr UserNotFoundError
		if errors.As(wrappedErr, &usernotfounderr) {
			fmt.Printf("user with %d id not found", usernotfounderr.UserID)
		}
	} else {
		fmt.Printf("user found:%s", name)
	}
}
