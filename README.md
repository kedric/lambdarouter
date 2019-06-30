This router is fully based on https://github.com/dimfeld/httptreemux
it was modified for use on AWS Lambda and take all advantage on this  

it's possible to use on local with mux server


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

	router.Serve(":8080")
}
```

## Serve
Serve method on router check if AWS_EXECUTION_ENV is set in env if is not set start mux server with port passed in argument else call lambda.Start(...) from aws sdk

on mux server all path has prefixed by ```/:__stage__```
when request oncomming the stage variable is stored in event.RequestContext.Stage 

## Stage Variables
if you need to pass a stageVariables to lambda with http handler use the header ```Stagevariable_{var_name}```
exemple: 
```
request header:
Stagevariable_foo=bar

lambda handler:
print(req.StageVariables["foo"]) // output: bar 
``` 

