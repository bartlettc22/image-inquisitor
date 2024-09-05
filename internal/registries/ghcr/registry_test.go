package ghcr

import "testing"

func TestParseLinkHeader(t *testing.T) {

	testCases := []struct {
		header   string
		expected string
	}{
		{`<https://example.com/api/items?page=2>; rel="next"`, "https://example.com/api/items?page=2"},
		{`<https://example.com/styles.css>; rel="preload"; as="style"`, ""},
		{`<https://example.com/article/123>; rel="canonical"`, ""},
		{`<https://m.example.com/article/123>; rel="alternate"; media="only screen and (max-width: 600px)"`, ""},
		{`<https://example.com/api/items?page=2>; rel="next", <https://example.com/api/items?page=1>; rel="prev", <https://example.com/styles.css>; rel="preload"; as="style"`,
			"https://example.com/api/items?page=2"},
	}

	for _, tc := range testCases {
		result := parseLinkHeaderRelNext(tc.header)

		if result != tc.expected {
			t.Errorf("link for `%s` not correct; want `%s`, got `%s`", tc.header, tc.expected, result)
		}
	}
}
