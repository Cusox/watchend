package store

import (
	"fmt"
)

func requirePositive(name string, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}
