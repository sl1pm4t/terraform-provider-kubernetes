package kubernetes

import (
	"fmt"
	"testing"
)

func TestIsInternalKey(t *testing.T) {
	testCases := []struct {
		Key      string
		Expected bool
	}{
		{"", false},
		{"anyKey", false},
		{"any.hostname.io", false},
		{"any.hostname.com/with/path", false},
		{"any.kubernetes.io", true},
		{"kubernetes.io", true},
		{"pv.kubernetes.io/any/path", true},
		{"kubernetes.io/egress-bandwidth", false},
		{"kubernetes.io/ingress.class", false},
		{"app.kubernetes.io/any/path", false},
		{"service.beta.kubernetes.io/any/path", false},
		{"nginx.ingress.kubernetes.io/any/path", false},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			isInternal := isInternalKey(tc.Key)
			if tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q to be internal", tc.Key)
			}
			if !tc.Expected && isInternal != tc.Expected {
				t.Fatalf("Expected %q not to be internal", tc.Key)
			}
		})
	}
}
