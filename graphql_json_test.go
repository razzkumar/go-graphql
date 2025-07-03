package graphql

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoJSON(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(b), `{"query":"query {}","variables":null}`+"\n"; got != want {
			t.Errorf("body got %v, want %v", got, want)
		}
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

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

func TestDoJSONServerError(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(b), `{"query":"query {}","variables":null}`+"\n"; got != want {
			t.Errorf("body got %v, want %v", got, want)
		}
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}
	if got, want := err.Error(), "graphql: server returned a non-200 status code: 500"; got != want {
		t.Errorf("err.Error() got %v, want %v", got, want)
	}
}

func TestDoJSONBadRequestErr(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Errorf("r.Method got %v, want %v", r.Method, http.MethodPost)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(b), `{"query":"query {}","variables":null}`+"\n"; got != want {
			t.Errorf("body got %v, want %v", got, want)
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
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if calls != 1 {
		t.Errorf("calls got %v, want %v", calls, 1)
	}
	if got, want := err.Error(), "graphql: miscellaneous message as to why the the request was bad"; got != want {
		t.Errorf("err.Error() got %v, want %v", got, want)
	}
}

func TestQueryJSON(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got, want := string(b), `{"query":"query {}","variables":{"username":"matryer"}}`+"\n"; got != want {
			t.Errorf("body got %v, want %v", got, want)
		}
		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

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

func TestHeader(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got, want := r.Header.Get("X-Custom-Header"), "123"; got != want {
			t.Errorf("r.Header.Get(\"X-Custom-Header\") got %v, want %v", got, want)
		}

		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Header.Set("X-Custom-Header", "123")

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
