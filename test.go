// build +dev

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stripe/stripe-cli/pkg/fixtures"
)

var filename string = "stripe"

func file(name string) string {
	return filepath.Join(".", name)
}

func build() error {
	cmd := exec.Command("make", "build")

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func cleanup() {
	os.Remove(file(filename))
}

func runTestTrigger(event string) error {
	cmd := exec.Command(filename, "trigger", event)

	out, err := cmd.Output()
	if err != nil {
		fmt.Println(string(out))
		return err
	}

	return nil
}

func testTrigger() error {
	for event := range fixtures.Events {
		fmt.Println(fmt.Sprintf("Running event: %s", event))

		err := runTestTrigger(event)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

func checkErr(err error) {
	if err != nil {
		cleanup()
		log.Fatalf("Test failure: %s", err)
	}
}

func main() {
	fmt.Println("Running end-to-end tests")
	fmt.Println("Building binary")

	err := build()
	checkErr(err)

	fmt.Println("Testing `trigger`")

	err = testTrigger()
	checkErr(err)

	cleanup()
}
