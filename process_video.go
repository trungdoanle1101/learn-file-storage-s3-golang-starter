package main

import "os/exec"

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmdName := "ffmpeg"
	execCmd := exec.Command(
		cmdName,
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputPath,
	)
	err := execCmd.Run()
	if err != nil {
		return "", err
	}
	return outputPath, nil
}

