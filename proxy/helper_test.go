package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/aws/aws-lambda-go/events"
)

func testHandler(context *RouteContext) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func testRequest(method HttpMethod, path string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		RawPath: path,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: method.String(),
			},
		},
		Headers: map[string]string{},
	}
}

func dummyNamespace(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return fmt.Sprintf("%s.*%s", t.Elem().PkgPath(), t.Elem().Name())
	} else {
		return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	}
}

func dummy(v interface{}, category string) interface{} {
	file := fmt.Sprintf("testdata/dummy/%s.%s.json", dummyNamespace(v), category)

	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(content, v)
	if err != nil {
		log.Fatal(err)
	}

	return v
}

func dummyAPIGatewayV2HTTPRequest(category string) events.APIGatewayV2HTTPRequest {
	return *dummy(&events.APIGatewayV2HTTPRequest{}, category).(*events.APIGatewayV2HTTPRequest)
}
