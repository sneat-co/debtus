package models4debtus

import (
	"errors"
	"fmt"
)

func ValidateString(errMess, s string, validValues []string) error {
	var ok bool
	for _, validValue := range validValues {
		if s == validValue {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("%v: '%v'", errMess, s)
	}
	return nil
}

var ErrNoProperties = errors.New("no properties")
