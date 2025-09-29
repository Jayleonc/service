package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return strings.TrimSpace(line), nil
}

func promptYesNo(reader *bufio.Reader, question string, defaultYes bool) (bool, error) {
	for {
		fmt.Print(question + " ")
		input, err := readLine(reader)
		if err != nil {
			return false, err
		}

		if input == "" {
			return defaultYes, nil
		}

		switch strings.ToLower(input) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer yes or no.")
		}
	}
}
