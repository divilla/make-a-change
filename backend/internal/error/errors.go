package error

import (
	"errors"
	"fmt"
)

const errorPrefix = "pgxpool:"

var (
	ExampleErr = errors.New("example error")
)

func ExampleError(err error) error {
	return fmt.Errorf("%s: %w", errorPrefix, fmt.Errorf("%w: %w", ExampleErr, err))
}
