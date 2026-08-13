package testfiles

import (
	"errors"
	"fmt"
)

func fixtures() {
	_ = errors.New("test leaf")
	_ = fmt.Errorf("test format")
}
