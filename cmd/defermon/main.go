package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/daohoangson/go-deferred/pkg/daemon"
)

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Printf("Usage: %s port secret\n", args[0])
		os.Exit(1)
	}

	port, err := strconv.ParseUint(args[1], 10, 16)
	if err != nil {
		fmt.Printf("Could not parse port %s (%s)\n", args[1], err)
		os.Exit(1)
	}

	d := daemon.New(nil, nil)
	d.SetSecret(args[2])
	d.ListenAndServe(port)
}
