package graphql

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWithClient(t *testing.T) {
	var calls int
	testClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			resp := &http.Response{
				Body: io.NopCloser(strings.NewReader(`{"data":{"key":"value"}}`)),
			}
			return resp, nil
		}),
	}

	ctx := context.Background()
	client := NewClient("", WithHTTPClient(testClient), UseMultipartForm())

	req := NewRequest(``)
	client.Run(ctx, req, nil)

	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}
}

func TestDoUseMultipartForm(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		query := r.FormValue("query")
		if query != `query {}` {
			t.Errorf("query got %v, want %v", query, `query {}`)
		}
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}
	if got, want := responseData["something"], "yes"; got != want {
		t.Errorf("responseData[\"something\"] got %v, want %v", got, want)
	}
}

func TestDoErr(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		query := r.FormValue("query")
		if query != `query {}` {
			t.Errorf("query got %v, want %v", query, `query {}`)
		}
		io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got, want := err.Error(), "graphql: Something went wrong"; got != want {
		t.Errorf("err.Error() got %v, want %v", got, want)
	}
}

func TestDoServerErr(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		query := r.FormValue("query")
		if query != `query {}` {
			t.Errorf("query got %v, want %v", query, `query {}`)
		}
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if got, want := err.Error(), "graphql: server returned a non-200 status code: 500"; got != want {
		t.Errorf("err.Error() got %v, want %v", got, want)
	}
}

func TestDoBadRequestErr(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		query := r.FormValue("query")
		if query != `query {}` {
			t.Errorf("query got %v, want %v", query, `query {}`)
		}
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{
			"errors": [{
				"message": "miscellaneous message as to why the the request was bad"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if got, want := err.Error(), "graphql: miscellaneous message as to why the the request was bad"; got != want {
		t.Errorf("err.Error() got %v, want %v", got, want)
	}
}

func TestDoNoResponse(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		query := r.FormValue("query")
		if query != `query {}` {
			t.Errorf("query got %v, want %v", query, `query {}`)
		}
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := client.Run(ctx, &Request{q: "query {}"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}
}

func TestQuery(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.FormValue("query")
		if query != "query {}" {
			t.Errorf("query got %v, want %v", query, "query {}")
		}
		if got, want := r.FormValue("variables"), `{"username":"matryer"}`+"\n"; got != want {
			t.Errorf("r.FormValue(\"variables\") got %v, want %v", got, want)
		}
		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL, UseMultipartForm())

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	if req == nil {
		t.Fatal("req should not be nil")
	}
	if got, want := req.vars["username"], "matryer"; got != want {
		t.Errorf("req.vars[\"username\"] got %v, want %v", got, want)
	}

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}

	if got, want := resp.Value, "some data"; got != want {
		t.Errorf("resp.Value got %v, want %v", got, want)
	}

}

func TestFile(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer file.Close()
		if header.Filename != "filename.txt" {
			t.Errorf("header.Filename got %v, want %v", header.Filename, "filename.txt")
		}

		b, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(b), `This is a file`; got != want {
			t.Errorf("string(b) got %v, want %v", got, want)
		}

		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	client := NewClient(srv.URL, UseMultipartForm())
	f := strings.NewReader(`This is a file`)
	req := NewRequest("query {}")
	req.File("file", "filename.txt", f)
	err := client.Run(ctx, req, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
