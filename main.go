package main

import (
	"github.com/triapex/auth/cmd/enroller"
	"time"
)

func main() {
	time.Local = nil
	cmd.Execute()
}
