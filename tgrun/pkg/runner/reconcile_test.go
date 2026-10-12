package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_matchCancelledInstance(t *testing.T) {
	cancelledIDs := map[string]bool{
		"abc123": true,
		"def456": true,
	}

	tests := []struct {
		name             string
		resourceName     string
		testIDAnnotation string
		wantID           string
		wantMatch        bool
	}{
		{
			name:             "annotation match",
			resourceName:     "abc123-initialprimary",
			testIDAnnotation: "abc123",
			wantID:           "abc123",
			wantMatch:        true,
		},
		{
			name:             "name prefix match without annotation",
			resourceName:     "def456-secondary-0",
			testIDAnnotation: "",
			wantID:           "def456",
			wantMatch:        true,
		},
		{
			name:             "sendlogs vmi name prefix match",
			resourceName:     "abc123-initialprimary-sendlogs",
			testIDAnnotation: "",
			wantID:           "abc123",
			wantMatch:        true,
		},
		{
			name:             "pvc name prefix match",
			resourceName:     "abc123-initialprimary-disk",
			testIDAnnotation: "",
			wantID:           "abc123",
			wantMatch:        true,
		},
		{
			name:             "unrelated vmi",
			resourceName:     "xyz789-initialprimary",
			testIDAnnotation: "xyz789",
			wantID:           "",
			wantMatch:        false,
		},
		{
			name:             "unrelated vmi without annotation",
			resourceName:     "xyz789-initialprimary",
			testIDAnnotation: "",
			wantID:           "",
			wantMatch:        false,
		},
		{
			name:             "instance id sharing a prefix does not match",
			resourceName:     "abc1234-initialprimary",
			testIDAnnotation: "abc1234",
			wantID:           "",
			wantMatch:        false,
		},
		{
			name:             "unrelated sendlogs vmi",
			resourceName:     "xyz789-initialprimary-sendlogs",
			testIDAnnotation: "",
			wantID:           "",
			wantMatch:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			id, ok := matchCancelledInstance(cancelledIDs, tt.resourceName, tt.testIDAnnotation)
			req.Equal(tt.wantMatch, ok)
			req.Equal(tt.wantID, id)
		})
	}
}
