//go:build wasm
// +build wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/stripe/stripe-cli/pkg/cmd"
)

func RunStripeCli() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ctx := context.Background()
		fmt.Println("Command received by WASM:")
		fmt.Println(args)
		cmd.PassInArgs(args)

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			go func() {
				// cmd.ExecuteWasm()
				cmd.Execute(ctx)
				resolve.Invoke()
			}()

			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}

func main() {
	wasmChan := make(chan bool)
	fmt.Println("Go Web Assembly From Stripe CLI!")
	js.Global().Set("StripeCliPromise", RunStripeCli())
	<-wasmChan
}
