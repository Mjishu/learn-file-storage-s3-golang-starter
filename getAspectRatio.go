package main

import (
	"encoding/json"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	if err != nil {
		return "", err
	}

	streams := result["streams"].([]interface{})
	if len(streams) == 0 {
		return "", nil
	}

	stream := streams[0].(map[string]interface{})
	width := stream["width"].(float64)
	height := stream["height"].(float64)
	// aspectRatio := width / height
	// do math to figure out how to get 16:9 or 9:16 if not either return other

	return string(output), nil

}
