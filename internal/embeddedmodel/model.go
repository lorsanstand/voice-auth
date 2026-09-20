package embeddedmodel

import (
	_ "embed"
	"errors"
	"os"
)

//go:embed speaker_model.onnx
var model []byte

func Materialize() (string, func(), error) {
	if len(model) == 0 {
		return "", func() {}, errors.New("embedded speaker model is empty")
	}

	file, err := os.CreateTemp("", "voice-auth-speaker-*.onnx")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}

	if _, err := file.Write(model); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return path, cleanup, nil
}
