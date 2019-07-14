This router is fully based on https://github.com/dimfeld/httptreemux
it was modified for use on AWS Lambda and take all advantage on this  

it's possible to use on local with builtin server 
NOT USE BUILTIN SERVER ON PRODUCTION


## Usage

```go
package main

import (
	"context"
	"fmt"
	"github.com/kedric/lambdarouter"
	"github.com/aws/aws-lambda-go/events"
	"log"
)

func Index(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error){
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:     "Welcome!\n",
	}, nil
}

func Hello(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("hello, %s!\n", req.PathParameters["name"]),
	}, nil
}

func main() {
	router := lambdarouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	router.Serve(":8080", nil)
}
```

## Serve
Serve method on router check if AWS_EXECUTION_ENV is set in env if is not set start mux server with port passed in argument else call lambda.Start(...) from aws sdk

on mux server all path has prefixed by ```/:__stage__```
when request oncomming the stage variable is stored in event.RequestContext.Stage 

## Stage Variables
if you need to pass a stageVariables to lambda with http handler add them on serv

exemple: 
```go
var variables = lambdarouter.StageVariables{
	"stagename": {
		"variablename":       "value",
	},
}

router.Serv(":8080", variables)
``` 

## Authorizer
ON PROGRESS

For use authorizer add handler function by router.SetAuthorizer(handler)
```go
func authorizer (ctx context.Context, request events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	...
}

func main() {
	router := lambdarouter.New()
	router.SetAuthorizer(authorizer)
	router.Serve(":8080", nil)
}
```
When you use builtin server it was call befor handler and passed on request.
When you deploy on lambda create spesific lambda with env variable AUTORIZER = true. 
