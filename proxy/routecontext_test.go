package proxy

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteContext_Body(t *testing.T) {
	request := testRequest(POST, "/yolo")
	request.Body = "some content"

	ctx := &RouteContext{Request: request}

	actual, err := ctx.Body()

	assert.NoError(t, err)
	assert.Equal(t, "some content", actual)
}

func TestRouteContext_Body_encoded(t *testing.T) {
	request := testRequest(POST, "/yolo")
	request.Body = base64.StdEncoding.EncodeToString([]byte("hey dude!"))
	request.IsBase64Encoded = true

	ctx := &RouteContext{Request: request}

	actual, err := ctx.Body()

	assert.NoError(t, err)
	assert.Equal(t, "hey dude!", actual)
}

func TestRouteContext_Body_error(t *testing.T) {
	request := testRequest(POST, "/yolo")
	request.Body = "sefdfxsdf.d.dsd"
	request.IsBase64Encoded = true

	ctx := &RouteContext{Request: request}

	_, err := ctx.Body()

	assert.Error(t, err)
}
