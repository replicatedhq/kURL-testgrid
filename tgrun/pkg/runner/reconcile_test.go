package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_isCancelledResource(t *testing.T) {
	cancelledIDs := map[string]bool{
		"abc123": true,
		"def456": true,
	}

	tests := []struct {
		name             string
		resourceName     string
		testIDAnnotation string
		want             bool
	}{
		{
			name:             "annotation match",
			resourceName:     "abc123-initialprimary",
			testIDAnnotation: "abc123",
			want:             true,
		},
		{
			name:             "name prefix match without annotation",
			resourceName:     "def456-secondary-0",
			testIDAnnotation: "",
			want:             true,
		},
		{
			name:             "sendlogs vmi name prefix match",
			resourceName:     "abc123-initialprimary-sendlogs",
			testIDAnnotation: "",
			want:             true,
		},
		{
			name:             "pvc name prefix match",
			resourceName:     "abc123-initialprimary-disk",
			testIDAnnotation: "",
			want:             true,
		},
		{
			name:             "unrelated vmi",
			resourceName:     "xyz789-initialprimary",
			testIDAnnotation: "xyz789",
			want:             false,
		},
		{
			name:             "unrelated vmi without annotation",
			resourceName:     "xyz789-initialprimary",
			testIDAnnotation: "",
			want:             false,
		},
		{
			name:             "instance id sharing a prefix does not match",
			resourceName:     "abc1234-initialprimary",
			testIDAnnotation: "abc1234",
			want:             false,
		},
		{
			name:             "unrelated sendlogs vmi",
			resourceName:     "xyz789-initialprimary-sendlogs",
			testIDAnnotation: "",
			want:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			req.Equal(tt.want, isCancelledResource(cancelledIDs, tt.resourceName, tt.testIDAnnotation))
		})
	}
}
