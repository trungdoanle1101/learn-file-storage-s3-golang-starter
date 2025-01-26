package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	type parameters struct {
		Streams []struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"streams"`
	}
	longRatio := float64(16) / float64(9)
	tallRatio := float64(9) / float64(16)
	cmdName := "ffprobe"
	execCmd := exec.Command(cmdName,
		"-v", "error",
		"-print_format", "json",
		"-show_streams", filePath)
	buffer := bytes.Buffer{}
	execCmd.Stdout = &buffer

	err := execCmd.Run()
	if err != nil {
		return "", err
	}
	var dim parameters

	err = json.Unmarshal(buffer.Bytes(), &dim)
	if err != nil {
		return "", err
	}

	width, height := dim.Streams[0].Width, dim.Streams[0].Height

	if width == 0.0 || height == 0.0 {
		return "other", nil
	}

	computedRatio := width / height

	if floatEquals(computedRatio, longRatio) {
		return "16:9", nil
	}

	if floatEquals(computedRatio, tallRatio) {
		return "9:16", nil
	}

	return "other", nil

}
func floatEquals(a, b float64) bool {
	return abs(a-b) < tol
}

const tol = 1e-2

func abs(x float64) float64 {
	if x < 0.0 {
		return -x
	}
	return x
}
