package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
)

func TestGetOperation(t *testing.T) {
	cases := []struct {
		name          string
		handler       http.Handler
		timeout       time.Duration
		retryInterval time.Duration
		wantErr       error
	}{
		{
			"happy path",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "text/json; charset=utf-8")
				fmt.Fprintln(w, `{"some": "json"}`)
			}),
			time.Millisecond * 200,
			time.Millisecond * 100,
			nil,
		},
		{
			"timeout",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				time.Sleep(time.Millisecond * 300)
				fmt.Fprintln(w, `{"some": "json"}`)
			}),
			time.Millisecond * 200,
			time.Millisecond * 100,
			fmt.Errorf("context deadline exceeded"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			_, err := backoff.Retry(ctx,
				GetOperation(ctx, t.TempDir(), ts.URL+"/data"),
				backoff.WithBackOff(backoff.NewConstantBackOff(tc.retryInterval)))
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				return
			}
			if err == nil {
				t.Fatalf("missing expected error result: got nil, wanted: %s", tc.wantErr.Error())
			}
			if tc.wantErr.Error() != err.Error() {
				t.Errorf("unexpected result: wanted '%s', got '%s'", tc.wantErr.Error(), err.Error())
			}
		})
	}
}
