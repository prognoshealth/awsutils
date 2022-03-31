package proxy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRoute(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	assert.Equal(t, GET, r.Method)
	assert.True(t, r.Regex.MatchString("/yolo"))
	assert.False(t, r.Regex.MatchString("/yolo/somethingelse"))
	assert.NotNil(t, r.Handler)
}

func TestNewRoute_Error(t *testing.T) {
	_, err := NewRoute(GET, "asom (?<in-invalid>.*)", testHandler)
	assert.Error(t, err)
}

func TestRoute_Match(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo"}, groups)
}

func TestRoute_wild(t *testing.T) {
	r, err := NewRoute(OPTIONS, ".*", testHandler)
	assert.NoError(t, err)

	request := testRequest(OPTIONS, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo"}, groups)
}

func TestRoute_Match_trailingSlash(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo/")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo/"}, groups)
}

func TestRoute_Match_groups(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<key>[^/]+)", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/yolo/the-id")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)
	assert.Equal(t, []string{"/yolo/the-id", "the-id"}, groups)
}

func TestRoute_Match_nope(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/something-else")
	matched, groups := r.IsMatch(request)

	assert.False(t, matched)
	assert.Nil(t, groups)
}

func TestRoute_Match_nopeMethod(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.False(t, matched)
	assert.Nil(t, groups)
}

func TestRoute_Context(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Empty(t, rctx.Params)
}

func TestRoute_Context_wild(t *testing.T) {
	r, err := NewRoute(OPTIONS, ".*", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(OPTIONS, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Empty(t, rctx.Params)
}

func TestRoute_Context_params_regex(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<id>[0-9]+)", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo/4")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"id": "4",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params_regex2(t *testing.T) {
	r, err := NewRoute(GET, "/yolo/(?P<id>[0-9]+)/fetch/(?P<state>[^/]+)", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo/4/fetch/ny")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"id":    "4",
		"state": "ny",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params_query(t *testing.T) {
	r, err := NewRoute(GET, "/wowza", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := dummyAPIGatewayV2HTTPRequest("params-query")

	matched, groups := r.IsMatch(request)
	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"jones": "arm",
		"whoop": "poor",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params_form(t *testing.T) {
	r, err := NewRoute(POST, "/wowza", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := dummyAPIGatewayV2HTTPRequest("params-form")

	matched, groups := r.IsMatch(request)
	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"dude": "the dude",
		"wow":  "scooby",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params_awspath(t *testing.T) {
	r, err := NewRoute(GET, "/wowza", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := dummyAPIGatewayV2HTTPRequest("params-awspath")

	matched, groups := r.IsMatch(request)
	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"jones": "leg",
		"whoop": "mistake",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_Context_params_multi(t *testing.T) {
	r, err := NewRoute(POST, "/wowza/(?P<regex>[^/]+)", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := dummyAPIGatewayV2HTTPRequest("params-multi")

	matched, groups := r.IsMatch(request)
	assert.True(t, matched)

	rctx, err := r.Context(ctx, request, groups)

	expected := map[string]string{
		"form":      "hi",
		"regex":     "hi",
		"query":     "hi",
		"awsparams": "hi",
	}

	assert.NoError(t, err)
	assert.Equal(t, ctx, rctx.Context)
	assert.Equal(t, request, rctx.Request)
	assert.Equal(t, expected, rctx.Params)
}

func TestRoute_extractParamsFromFormPost_not_post(t *testing.T) {
	r, err := NewRoute(GET, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(GET, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"

	params := map[string]string{}
	expected := map[string]string{}

	err = r.extractParamsFromFormPost(params, request)

	assert.NoError(t, err)
	assert.Equal(t, expected, params)
}

func TestRoute_extractParamsFromFormPost_not_form(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "text/plain"

	params := map[string]string{}
	expected := map[string]string{}

	err = r.extractParamsFromFormPost(params, request)

	assert.NoError(t, err)
	assert.Equal(t, expected, params)
}

func TestRoute_extractParamsFromFormPost_base64(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"
	request.IsBase64Encoded = true
	request.Body = "eW9sbz1kaWNlJnN1cGVyPXNtYXJ0eStwYW50eg=="

	params := map[string]string{}
	expected := map[string]string{
		"super": "smarty pantz",
		"yolo":  "dice",
	}

	err = r.extractParamsFromFormPost(params, request)

	assert.NoError(t, err)
	assert.Equal(t, expected, params)
}

func TestRoute_extractParamsFromFormPost_nobase64(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"
	request.IsBase64Encoded = false
	request.Body = "super=red+sonya&die=hard"

	params := map[string]string{}
	expected := map[string]string{
		"super": "red sonya",
		"die":   "hard",
	}

	err = r.extractParamsFromFormPost(params, request)

	assert.NoError(t, err)
	assert.Equal(t, expected, params)
}

func TestRoute_extractParamsFromFormPost_error_base64(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"
	request.IsBase64Encoded = true
	request.Body = "eW9sbz1ka****=="

	params := map[string]string{}

	err = r.extractParamsFromFormPost(params, request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "illegal base64 data")
}

func TestRoute_extractParamsFromFormPost_error_form(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"
	request.IsBase64Encoded = false
	request.Body = "asdfg=qrr&sas"

	params := map[string]string{}

	err = r.extractParamsFromFormPost(params, request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key/value pair")
}

func TestRoute_extractParamsFromFormPost_error_encode(t *testing.T) {
	r, err := NewRoute(POST, "/hi", testHandler)
	assert.NoError(t, err)

	request := testRequest(POST, "/hi")
	request.Headers["content-type"] = "application/x-www-form-urlencoded"
	request.IsBase64Encoded = false
	request.Body = "asdfg=hi %Z yolo"

	params := map[string]string{}

	err = r.extractParamsFromFormPost(params, request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to decode")
}

func TestRoute_Follow(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")
	matched, groups := r.IsMatch(request)

	assert.True(t, matched)

	response, err := r.Follow(ctx, request, groups)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
}

func TestRoute_Follow_error(t *testing.T) {
	r, err := NewRoute(GET, "/yolo", testHandler)
	assert.NoError(t, err)

	ctx := context.Background()
	request := testRequest(GET, "/yolo")

	_, err = r.Follow(ctx, request, []string{})
	assert.Error(t, err)
}
