package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func ConvertIntoFrames(videoName string) {
	fmt.Println("Slicing the video into frames...")
	cmd := exec.Command("util/slice.sh", strings.Split(videoName, ".")[0])

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal("slice.sh cmd execution error:", err)
	}
	fmt.Println("Slicing complete.")
}
