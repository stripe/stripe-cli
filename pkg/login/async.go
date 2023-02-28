package login

import (
	"fmt"
	"os"
)

// AsyncInputReader is an interface that has an async version of scanln
type AsyncInputReader interface {
	scanln(ch chan int)
}

// AsyncStdinReader implements scanln(ch chan int), an async version of scanln
type AsyncStdinReader struct {
}

func (r AsyncStdinReader) scanln(ch chan int) {
	n, _ := fmt.Fscanln(os.Stdin)
	ch <- n
}
