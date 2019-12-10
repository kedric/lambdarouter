package lambdarouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/aws/aws-lambda-go/events"
)

func LambdaGenerateRawQuery(request events.APIGatewayProxyRequest) string {
	tmp := url.Values{}
	for i := range request.QueryStringParameters {
		tmp.Add(i, request.QueryStringParameters[i])
	}
	return tmp.Encode()
}

func LambdaRedirect(ctx context.Context, req events.APIGatewayProxyRequest, newUrl string, code int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Location": newUrl,
		},
	}, nil
}

func LambdaAllow(ctx context.Context, req events.APIGatewayProxyRequest, allow string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Allow": allow,
		},
	}, nil
}

func LambdaNotAllowed(ctx context.Context, req events.APIGatewayProxyRequest, allow string) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 405,
		Headers: map[string]string{
			"Allow": allow,
		},
		Body: `{"error": "Method Not Allowed"}`,
	}, nil
}

func LambdaNotFound(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 404,
		Body:       `{"error": "Not Found"}`,
	}, nil
}

func GetForwarded(r *http.Request) string {
	var remoteIP string
	if strings.ContainsRune(r.RemoteAddr, ':') {
		remoteIP, _, _ = net.SplitHostPort(r.RemoteAddr)
	} else {
		remoteIP = r.RemoteAddr
	}
	return strings.Trim(fmt.Sprintf("%s,%s", r.Header.Get("X-Forwarded-For"), remoteIP), " ,")
}

func RequestToLambda(req *http.Request) (events.APIGatewayProxyRequest, error) {
	e := events.APIGatewayProxyRequest{
		HTTPMethod:            req.Method,
		Path:                  strings.Split(req.URL.RequestURI(), "?")[0],
		Resource:              strings.Split(req.URL.RequestURI(), "?")[0],
		Headers:               map[string]string{},
		QueryStringParameters: map[string]string{},
		PathParameters:        map[string]string{},
		StageVariables:        map[string]string{},
	}
	// e.RequestContext.RequestID = utils.UUID()
	// e.RequestContext.ResourcePath = params.Path
	e.RequestContext.HTTPMethod = req.Method
	for i := range req.URL.Query() {

		e.QueryStringParameters[i] = req.URL.Query().Get(i)
	}
	for i := range req.Header {
		e.Headers[i] = req.Header.Get(i)
	}
	e.Headers["X-Forwarded-For"] = GetForwarded(req)
	if req.Body != nil {
		b, _ := ioutil.ReadAll(req.Body)
		e.Body = fmt.Sprintf("%s", b)
	}
	return e, nil
}

func ResToHttp(w http.ResponseWriter, req *http.Request, res events.APIGatewayProxyResponse) {
	for key := range res.Headers {
		w.Header().Set(key, res.Headers[key])
	}
	w.WriteHeader(res.StatusCode)
	if res.IsBase64Encoded {
		data, err := base64.StdEncoding.DecodeString(res.Body)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("Error on decoding base64: %s\n", err.Error())))
			return
		}
		w.Write(data)
		return
	}
	w.Write([]byte(res.Body))
}

func HttpAddParams(event events.APIGatewayProxyRequest, params map[string]string) events.APIGatewayProxyRequest {
	event.PathParameters = params
	return event
}

func UseTemplate(event events.APIGatewayProxyRequest) string {
	tmpResource := strings.ReplaceAll(event.Resource, "{", "{{.")
	tmpResource = strings.ReplaceAll(tmpResource, "}", "}}")
	tmpResource = strings.ReplaceAll(tmpResource, "+", "")
	tmpl, err := template.New("route").Parse(tmpResource)
	if err != nil {
		return event.Path
	}
	out := bytes.NewBuffer([]byte{})
	tmpl.Execute(out, event.PathParameters)
	return string(out.Bytes())
}

func CleanPath(event events.APIGatewayProxyRequest) string {
	return UseTemplate(event)
}

func GenerateArn(event events.APIGatewayProxyRequest) string {
	return fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*/%s/%s", os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCOUNT_ID"), "localhost", event.HTTPMethod, event.Path)
}

func GenerateLambdaAuthorizer(event events.APIGatewayProxyRequest) events.APIGatewayCustomAuthorizerRequestTypeRequest {
	return events.APIGatewayCustomAuthorizerRequestTypeRequest{
		MethodArn:                       GenerateArn(event),
		Path:                            event.Path,
		HTTPMethod:                      event.HTTPMethod,
		Headers:                         event.Headers,
		MultiValueHeaders:               event.MultiValueHeaders,
		QueryStringParameters:           event.QueryStringParameters,
		MultiValueQueryStringParameters: event.MultiValueQueryStringParameters,
		PathParameters:                  event.PathParameters,
		StageVariables:                  event.StageVariables,
	}
}

type EnumEventType int

const (
	NotFound EnumEventType = iota
	Http
	Authorizer
	Websocket
)

func mapHaveKeys(_map map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := _map[key]; !ok {
			return false
		}
	}
	return true
}

func GetEventType(ctx context.Context, event map[string]interface{}) EnumEventType {
	tmp, _ := json.Marshal(event)
	fmt.Printf("%s\n", tmp)
	if mapHaveKeys(event, "type") {
		return Authorizer
	} else {
		if isWebsocketEvent(event) {
			return Websocket
		}
		return Http
	}
	return NotFound
}

func toHttpEvent(event map[string]interface{}) events.APIGatewayProxyRequest {
	tmp, _ := json.Marshal(event)
	ret := events.APIGatewayProxyRequest{}
	json.Unmarshal(tmp, &ret)
	return ret
}

func toWsEvent(event map[string]interface{}) events.APIGatewayWebsocketProxyRequest {
	tmp, _ := json.Marshal(event)
	ret := events.APIGatewayWebsocketProxyRequest{}
	json.Unmarshal(tmp, &ret)
	return ret
}

func toAuthorizerEvent(event map[string]interface{}) events.APIGatewayCustomAuthorizerRequestTypeRequest {
	tmp, _ := json.Marshal(event)
	ret := events.APIGatewayCustomAuthorizerRequestTypeRequest{}
	json.Unmarshal(tmp, &ret)
	return ret
}
