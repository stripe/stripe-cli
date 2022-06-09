//go:build wasm
// +build wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/stripe/stripe-cli/pkg/cmd"
)

func executeCommandWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx := context.Background()
		cmd.PassInArgs(args)
		cmd.Execute(ctx)
		return nil
	})
}

func main() {
	fmt.Println("Go Web Assembly From Stripe CLI!")
	js.Global().Set("stripeCli", executeCommandWrapper())
	<-make(chan bool)
}
