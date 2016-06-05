package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	comPort := os.Args[1]

	for {
		cmd := exec.Command("server.exe", comPort)
		stdout, _ := cmd.StdoutPipe()
		cmd.Start()
		for {
			b := make([]byte, 1, 1)
			if _, err := stdout.Read(b); err == nil {
				fmt.Print(string(b))
			} else {
				break
			}
		}
		cmd.Wait()
	}
}
