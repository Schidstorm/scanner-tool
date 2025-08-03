package server

import (
	"encoding/base64"
	"testing"

	queueoutputcreator "github.com/schidstorm/scanner-tool/pkg/queue_output_creator"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestImageMirrorHandler(t *testing.T) {
	var normal = "iVBORw0KGgoAAAANSUhEUgAAABcAAAAYCAYAAAARfGZ1AAAACXBIWXMAAA7EAAAOxAGVKw4bAAAAc0lEQVRIie2VMQrAIAwAz9J/6c80LzO+LJ3aoWiRgEPBg0zBMxATg5kZizhWibd8yOk5JCKf+Zwz4KhcRKi1YmbdUFVKKYCz8pTSI+hdfr/u/zZ0y7d8HtcQtdaGK0BViTECEDyfxWg633mXfJb/NnTLu1xBvTkZ1JAGlwAAAABJRU5ErkJggg=="
	var mirrored = "iVBORw0KGgoAAAANSUhEUgAAABcAAAAYCAIAAACeHvEiAAAAbklEQVR4nOySMQ7FIAxDP18cDE6GuZl9MjrQqUBFlSUDXpCI8hRZL7bWfub87YhDWSb2B8B0vPqf3AJAUhgiqdb64ZaUUinlMdtXyVO7h+KfcltHcnSMZM55hxL68sr0Ueg3ijGe2vVEuQIAAP//DkQkQ67bYFoAAAAASUVORK5CYII="

	normalImageData, _ := base64.StdEncoding.DecodeString(normal)
	mirroredImageData, _ := base64.StdEncoding.DecodeString(mirrored)

	resultFiles := prepareHandlerThings(t, map[string][]byte{
		"normal.png": normalImageData,
	}, func(input chan InputFile, outputFiles queueoutputcreator.QueueZipFileWriter) error {
		handler := &ImageMirrorHandler{}
		return handler.Run(logrus.New(), input, outputFiles)
	})

	assert.Equal(t, 1, len(resultFiles))
	assert.Equal(t, base64.StdEncoding.EncodeToString(mirroredImageData), base64.StdEncoding.EncodeToString(resultFiles["normal.png"]))
}
