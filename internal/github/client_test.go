package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOwnerRepos(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/example/repos":
			// Two pages: 100 repos, then one
			if r.URL.Query().Get("page") == "1" {
				fmt.Fprint(w, "[")
				for i := 0; i < pageSize; i++ {
					if i > 0 {
						fmt.Fprint(w, ",")
					}
					fmt.Fprintf(w, `{"full_name":"example/r%d","owner":{"login":"example"}}`, i)
				}
				fmt.Fprint(w, "]")
				return
			}
			fmt.Fprint(w, `[{"full_name":"example/last","archived":true,"owner":{"login":"example"}}]`)
		case "/user/repos":
			fmt.Fprint(w, `[{"full_name":"chris/private","owner":{"login":"chris"}},{"full_name":"example/x","owner":{"login":"example"}}]`)
		case "/users/someone/repos":
			fmt.Fprint(w, `[{"full_name":"someone/public","fork":true,"owner":{"login":"someone"}}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "token")

	repos, err := c.ListOwnerRepos(context.Background(), "example")
	require.NoError(t, err)
	require.Len(t, repos, pageSize+1)
	assert.True(t, repos[pageSize].Archived)

	repos, err = c.ListOwnerRepos(context.Background(), "chris")
	require.NoError(t, err)
	assert.Equal(t, "chris/private", repos[0].FullName, "the token's own user, including private repos")
	assert.Len(t, repos, 1)

	repos, err = c.ListOwnerRepos(context.Background(), "someone")
	require.NoError(t, err)
	assert.Equal(t, "someone/public", repos[0].FullName)
	assert.True(t, repos[0].Fork)
}
