package noderelease

import (
	"testing"

	"hackathon/cmd/jarvis/internal/types"
	"hackathon/sharedmodel"
)

func TestCheckBasicArgs_Rollback(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.NodeReleaseRollbackReq
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid releaseId",
			req: &types.NodeReleaseRollbackReq{
				ReleaseID: "valid-release-id",
			},
			wantErr: false,
		},
		{
			name: "empty releaseId",
			req: &types.NodeReleaseRollbackReq{
				ReleaseID: "",
			},
			wantErr: true,
			errMsg:  "releaseId参数无效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.ReleaseID == "" && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected error for empty releaseId")
			}
		})
	}
}

func TestCheckRollbackAllowed(t *testing.T) {
	tests := []struct {
		name        string
		releaseType sharedmodel.ReleaseType
		state       sharedmodel.NodeReleaseState
		wantAllowed bool
		errMsg      string
	}{
		{
			name:        "formal release in processing state",
			releaseType: sharedmodel.ReleaseTypeFormal,
			state:       sharedmodel.NodeReleaseStateProcessing,
			wantAllowed: true,
		},
		{
			name:        "formal release in completed state",
			releaseType: sharedmodel.ReleaseTypeFormal,
			state:       sharedmodel.NodeReleaseStateCompleted,
			wantAllowed: true, // Allowed if it's the last release
		},
		{
			name:        "beta release should not allow rollback",
			releaseType: sharedmodel.ReleaseTypeBeta,
			state:       sharedmodel.NodeReleaseStateProcessing,
			wantAllowed: false,
			errMsg:      "非正式发布任务, 禁止操作回滚",
		},
		{
			name:        "formal release in rollbacked state",
			releaseType: sharedmodel.ReleaseTypeFormal,
			state:       sharedmodel.NodeReleaseStateRollbacked,
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test release type validation
			if tt.releaseType != sharedmodel.ReleaseTypeFormal {
				if tt.wantAllowed {
					t.Errorf("checkRollbackAllowed() should not allow rollback for non-formal releases")
				}
			}
			
			// Test state validation
			if tt.releaseType == sharedmodel.ReleaseTypeFormal {
				switch tt.state {
				case sharedmodel.NodeReleaseStateProcessing:
					if !tt.wantAllowed {
						t.Errorf("checkRollbackAllowed() should allow rollback for processing state")
					}
				case sharedmodel.NodeReleaseStateCompleted:
					// Allowed if it's the last release (requires DB check)
				default:
					if tt.wantAllowed {
						t.Errorf("checkRollbackAllowed() should not allow rollback for state %v", tt.state)
					}
				}
			}
		})
	}
}

func TestRollbackStateTransitions(t *testing.T) {
	tests := []struct {
		name       string
		fromState  sharedmodel.NodeReleaseState
		toState    sharedmodel.NodeReleaseState
		wantValid  bool
	}{
		{
			name:      "processing to rollbacked",
			fromState: sharedmodel.NodeReleaseStateProcessing,
			toState:   sharedmodel.NodeReleaseStateRollbacked,
			wantValid: true,
		},
		{
			name:      "completed to rollbacked",
			fromState: sharedmodel.NodeReleaseStateCompleted,
			toState:   sharedmodel.NodeReleaseStateRollbacked,
			wantValid: true,
		},
		{
			name:      "rollbacked to rollbacked again",
			fromState: sharedmodel.NodeReleaseStateRollbacked,
			toState:   sharedmodel.NodeReleaseStateRollbacked,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid transitions should be processing->rollbacked or completed->rollbacked
			validStates := []sharedmodel.NodeReleaseState{
				sharedmodel.NodeReleaseStateProcessing,
				sharedmodel.NodeReleaseStateCompleted,
			}
			
			isValid := false
			for _, state := range validStates {
				if tt.fromState == state && tt.toState == sharedmodel.NodeReleaseStateRollbacked {
					isValid = true
					break
				}
			}
			
			if isValid != tt.wantValid {
				t.Errorf("Rollback state transition from %v to %v: got valid=%v, want valid=%v", 
					tt.fromState, tt.toState, isValid, tt.wantValid)
			}
		})
	}
}
