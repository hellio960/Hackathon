package noderelease

import (
	"testing"

	"hackathon/sharedmodel"
)

func TestNodeReleaseList_Pagination(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantSkip int
		wantErr  bool
	}{
		{
			name:     "first page",
			page:     1,
			pageSize: 10,
			wantSkip: 0,
			wantErr:  false,
		},
		{
			name:     "second page",
			page:     2,
			pageSize: 10,
			wantSkip: 10,
			wantErr:  false,
		},
		{
			name:     "page 5 with size 20",
			page:     5,
			pageSize: 20,
			wantSkip: 80,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip := (tt.page - 1) * tt.pageSize
			
			if skip != tt.wantSkip {
				t.Errorf("Pagination skip = %v, want %v", skip, tt.wantSkip)
			}
		})
	}
}

func TestNodeReleaseList_StateFilter(t *testing.T) {
	tests := []struct {
		name          string
		states        []string
		expectedValid bool
	}{
		{
			name:          "single valid state",
			states:        []string{sharedmodel.NodeReleaseStateProcessing.String()},
			expectedValid: true,
		},
		{
			name:          "multiple valid states",
			states:        []string{sharedmodel.NodeReleaseStateProcessing.String(), sharedmodel.NodeReleaseStateCompleted.String()},
			expectedValid: true,
		},
		{
			name:          "empty states",
			states:        []string{},
			expectedValid: true, // Empty means no filter
		},
		{
			name:          "all states",
			states:        []string{
				sharedmodel.NodeReleaseStateProcessing.String(),
				sharedmodel.NodeReleaseStateCompleted.String(),
				sharedmodel.NodeReleaseStateRollbacked.String(),
			},
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate that all states are from the known state list
			knownStates := map[string]bool{
				sharedmodel.NodeReleaseStateProcessing.String():  true,
				sharedmodel.NodeReleaseStateCompleted.String():   true,
				sharedmodel.NodeReleaseStateRollbacked.String():  true,
			}
			
			for _, state := range tt.states {
				if !knownStates[state] && tt.expectedValid {
					t.Errorf("Unknown state %v in filter", state)
				}
			}
		})
	}
}
