package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func HLSConverter() {
	fmt.Println("Starting HLS conversion...")
	cmd := exec.Command("util/convert-to-hls.sh")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal("convert-to-hls.sh cmd execution error:", err)
	}
	fmt.Println("HLS conversion complete.")
}

func DASHConverter() {
	fmt.Println("Starting MPEG-DASH conversion...")
	cmd := exec.Command("util/convert-to-dash.sh", "all")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal("convert-to-dash.sh cmd execution error:", err)
	}
	fmt.Println("MPEG-DASH conversion complete.")
}
