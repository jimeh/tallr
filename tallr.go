package main

import (
	"errors"
	"fmt"
)

var (
	Err              = errors.New("tallr")
	ErrAlreadyParsed = fmt.Errorf("%w: already parsed", Err)
)
