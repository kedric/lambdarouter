// +build go1.7

package lambdarouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

type IContextGroup interface {
	GET(path string, handler HandlerFunc)
	POST(path string, handler HandlerFunc)
	PUT(path string, handler HandlerFunc)
	PATCH(path string, handler HandlerFunc)
	DELETE(path string, handler HandlerFunc)
	HEAD(path string, handler HandlerFunc)
	OPTIONS(path string, handler HandlerFunc)

	NewContextGroup(path string) *ContextGroup
	NewGroup(path string) *ContextGroup
}

func TestContextParams(t *testing.T) {
	m := map[string]string{"id": "123"}
	ctx := context.WithValue(context.Background(), paramsContextKey, m)

	params := ContextParams(ctx)
	if params == nil {
		t.Errorf("expected '%#v', but got '%#v'", m, params)
	}

	if v := params["id"]; v != "123" {
		t.Errorf("expected '%s', but got '%#v'", m["id"], params["id"])
	}
}

func TestContextGroupMethods(t *testing.T) {
	for _, scenario := range scenarios {
		t.Log(scenario.description)
		testContextGroupMethods(t, scenario.RequestCreator, true, false)
		testContextGroupMethods(t, scenario.RequestCreator, false, false)
		testContextGroupMethods(t, scenario.RequestCreator, true, true)
		testContextGroupMethods(t, scenario.RequestCreator, false, true)
	}
}

func testContextGroupMethods(t *testing.T, reqGen RequestCreator, headCanUseGet bool, useContextRouter bool) {
	t.Logf("Running test: headCanUseGet %v, useContextRouter %v", headCanUseGet, useContextRouter)

	var result string
	makeHandler := func(method string) HandlerFunc {
		return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			result = method
			v, ok := req.PathParameters["param"]
			if !ok {
				t.Error("missing key 'param' in context")
			}

			if headCanUseGet && (req.HTTPMethod == "GET" || v == "HEAD") {
				return events.APIGatewayProxyResponse{StatusCode: 200, Body: v}, nil
			}

			if v != method {
				t.Errorf("invalid key 'param' in context; expected '%s' but got '%s'", method, v)
			}
			return events.APIGatewayProxyResponse{StatusCode: 400}, nil
		}
	}

	var router http.Handler
	var rootGroup IContextGroup

	if useContextRouter {
		root := NewContextMux()
		root.HeadCanUseGet = headCanUseGet
		t.Log(root.TreeMux.HeadCanUseGet)
		router = root
		rootGroup = root
	} else {
		root := New()
		root.HeadCanUseGet = headCanUseGet
		router = root
		rootGroup = root.UsingContext()
	}

	cg := rootGroup.NewGroup("/base").NewGroup("/user")
	cg.GET("/:param", makeHandler("GET"))
	cg.POST("/:param", makeHandler("POST"))
	cg.PATCH("/:param", makeHandler("PATCH"))
	cg.PUT("/:param", makeHandler("PUT"))
	cg.DELETE("/:param", makeHandler("DELETE"))

	testMethod := func(method, expect string) {
		result = ""
		w := httptest.NewRecorder()
		r, _ := reqGen(method, "/__stage__/base/user/"+method, nil)
		router.ServeHTTP(w, r)
		if expect == "" && w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Method %s not expected to match but saw code %d", method, w.Code)
		}

		if result != expect {
			t.Errorf("Method %s got result %s", method, result)
		}
	}

	testMethod("GET", "GET")
	testMethod("POST", "POST")
	testMethod("PATCH", "PATCH")
	testMethod("PUT", "PUT")
	testMethod("DELETE", "DELETE")

	if headCanUseGet {
		t.Log("Test implicit HEAD with HeadCanUseGet = true")
		testMethod("HEAD", "GET")
	} else {
		t.Log("Test implicit HEAD with HeadCanUseGet = false")
		testMethod("HEAD", "")
	}

	cg.HEAD("/:param", makeHandler("HEAD"))
	testMethod("HEAD", "HEAD")
}

func TestNewContextGroup(t *testing.T) {
	router := New()
	group := router.NewGroup("/api")

	group.GET("/v1", func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: `200 OK GET /api/v1`}, nil
	})

	group.UsingContext().GET("/v2", func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{StatusCode: 200, Body: `200 OK GET /api/v2`}, nil
	})

	tests := []struct {
		uri, expected string
	}{
		{"/__stage__/api/v1", "200 OK GET /api/v1"},
		{"/__stage__/api/v2", "200 OK GET /api/v2"},
	}

	for _, tc := range tests {
		r, err := http.NewRequest("GET", tc.uri, nil)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("GET %s: expected %d, but got %d", tc.uri, http.StatusOK, w.Code)
		}
		if got := w.Body.String(); got != tc.expected {
			t.Errorf("GET %s : expected %q, but got %q", tc.uri, tc.expected, got)
		}

	}
}

type ContextGroupHandler struct{}

//	adhere to the http.Handler interface
func (f ContextGroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Write([]byte(`200 OK GET /api/v1`))
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

// func TestNewContextGroupHandler(t *testing.T) {
// 	router := New()
// 	group := router.NewGroup("/api")

// 	group.UsingContext().Handler("GET", "/v1", ContextGroupHandler{})

// 	tests := []struct {
// 		uri, expected string
// 	}{
// 		{"/api/v1", "200 OK GET /api/v1"},
// 	}

// 	for _, tc := range tests {
// 		r, err := http.NewRequest("GET", tc.uri, nil)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		w := httptest.NewRecorder()
// 		router.ServeHTTP(w, r)

// 		if w.Code != http.StatusOK {
// 			t.Errorf("GET %s: expected %d, but got %d", tc.uri, http.StatusOK, w.Code)
// 		}
// 		if got := w.Body.String(); got != tc.expected {
// 			t.Errorf("GET %s : expected %q, but got %q", tc.uri, tc.expected, got)
// 		}
// 	}
// }

// func TestDefaultContext(t *testing.T) {
// 	router := New()
// 	ctx := context.WithValue(context.Background(), "abc", "def")
// 	expectContext := false

// 	router.GET("/abc",  func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// 		contextValue := r.Context().Value("abc")
// 		if expectContext {
// 			x, ok := contextValue.(string)
// 			if !ok || x != "def" {
// 				t.Errorf("Unexpected context key value: %+v", contextValue)
// 			}
// 		} else {
// 			if contextValue != nil {
// 				t.Errorf("Expected blank context but key had value %+v", contextValue)
// 			}
// 		}
// 	})

// 	r, err := http.NewRequest("GET", "/abc", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	w := httptest.NewRecorder()
// 	t.Log("Testing without DefaultContext")
// 	router.ServeHTTP(w, r)

// 	router.DefaultContext = ctx
// 	expectContext = true
// 	w = httptest.NewRecorder()
// 	t.Log("Testing with DefaultContext")
// 	router.ServeHTTP(w, r)
// }

// func TestContextMuxSimple(t *testing.T) {
// 	router := NewContextMux()
// 	ctx := context.WithValue(context.Background(), "abc", "def")
// 	expectContext := false

// 	router.GET("/abc", func(w http.ResponseWriter, r *http.Request) {
// 		contextValue := r.Context().Value("abc")
// 		if expectContext {
// 			x, ok := contextValue.(string)
// 			if !ok || x != "def" {
// 				t.Errorf("Unexpected context key value: %+v", contextValue)
// 			}
// 		} else {
// 			if contextValue != nil {
// 				t.Errorf("Expected blank context but key had value %+v", contextValue)
// 			}
// 		}
// 	})

// 	r, err := http.NewRequest("GET", "/abc", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	w := httptest.NewRecorder()
// 	t.Log("Testing without DefaultContext")
// 	router.ServeHTTP(w, r)

// 	router.DefaultContext = ctx
// 	expectContext = true
// 	w = httptest.NewRecorder()
// 	t.Log("Testing with DefaultContext")
// 	router.ServeHTTP(w, r)
// }
