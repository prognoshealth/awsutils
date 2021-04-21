// Package proxy provides utilities for writing aws lambda functions that act as
// aws api gateway v2 (http) integrations. Specifically they assist in adding
// routing functionality and processing the entire request/response through the
// lambda via events.APIGatewayV2HTTPRequest and events.APIGatewayProxyResponse.
//
// The router is designed to be as simplistic as possible and is not feature
// rich.
package proxy
