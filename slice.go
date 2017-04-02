package main

import (
	"bufio"
	"io"
	"os"

	"github.com/urfave/cli"
)

func slice(input io.Reader, from, to int, output *bufio.Writer) error {
	if to != -1 && (to <= from) {
		os.Stderr.WriteString("--from argument must be greater than --to argument\n")
		return USAGE_ERROR
	}
	if from < 0 {
		os.Stderr.WriteString("--from must be greater than or equal to 0\n")
		return USAGE_ERROR
	}

	pending := make(chan string, BUFFER_SIZE)
	ret := make(chan error)

	inputs := []io.Reader{input}
	go readInputs(inputs, pending, ret)
	err, ok := <-ret
	if ok {
		return err
	}

	i := 0
	for i != from {
		_, ok := <-pending
		if !ok {
			return cli.NewExitError("slice beginning not reached", 2)
		}
		i++
	}
	handleLines(output, pending, to-from)
	return nil
}
