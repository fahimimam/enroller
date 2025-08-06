package main

import (
	"github.com/triapex/auth/cmd/auth"
	"time"
)

func main() {
	time.Local = nil
	cmd.Execute()
}
