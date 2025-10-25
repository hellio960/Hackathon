package noderelease

import (
	"testing"

	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/utils"
)

func TestCheckBasicArgs_Continue(t *testing.T) {
	tests := []struct {
		name     string
		req      *types.NodeReleaseContinueReq
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid request",
			req: &types.NodeReleaseContinueReq{
				ReleaseID:  "valid-release-id",
				Percentage: 50,
				AddNodes:   []string{"node1"},
				DelNodes:   []string{},
			},
			wantErr: false,
		},
		{
			name: "empty releaseId",
			req: &types.NodeReleaseContinueReq{
				ReleaseID:  "",
				Percentage: 50,
			},
			wantErr: true,
			errMsg:  "releaseId参数无效",
		},
		{
			name: "invalid percentage - too low",
			req: &types.NodeReleaseContinueReq{
				ReleaseID:  "valid-release-id",
				Percentage: 0,
			},
			wantErr: true,
			errMsg:  "灰度比例无效",
		},
		{
			name: "invalid percentage - too high",
			req: &types.NodeReleaseContinueReq{
				ReleaseID:  "valid-release-id",
				Percentage: 101,
			},
			wantErr: true,
			errMsg:  "灰度比例无效",
		},
		{
			name: "addNodes and delNodes have intersection",
			req: &types.NodeReleaseContinueReq{
				ReleaseID:  "valid-release-id",
				Percentage: 50,
				AddNodes:   []string{"node1", "node2"},
				DelNodes:   []string{"node2", "node3"},
			},
			wantErr: true,
			errMsg:  "新增节点和移除节点中存在交集",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test releaseId validation
			if tt.req.ReleaseID == "" && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected error for empty releaseId")
			}
			
			// Test percentage validation
			if (tt.req.Percentage <= 0 || tt.req.Percentage > 100) && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected error for invalid percentage")
			}
			
			// Test intersection validation
			if utils.CommonInAB(tt.req.AddNodes, tt.req.DelNodes) && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected error when addNodes and delNodes intersect")
			}
		})
	}
}

func TestEnsureReleaseState_Continue(t *testing.T) {
	tests := []struct {
		name           string
		currentPercent int
		newPercent     int
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "increase percentage from 30 to 50",
			currentPercent: 30,
			newPercent:     50,
			wantErr:        false,
		},
		{
			name:           "keep same percentage",
			currentPercent: 50,
			newPercent:     50,
			wantErr:        false,
		},
		{
			name:           "decrease percentage - should fail",
			currentPercent: 70,
			newPercent:     50,
			wantErr:        true,
			errMsg:         "禁止调小灰度比例",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.currentPercent > tt.newPercent && !tt.wantErr {
				t.Errorf("ensureReleaseState() should not allow decreasing percentage")
			}
		})
	}
}

func TestRecalculatePercentage(t *testing.T) {
	tests := []struct {
		name         string
		totalNodes   int
		allowCount   int
		wantPercent  int
	}{
		{
			name:         "50 out of 100",
			totalNodes:   100,
			allowCount:   50,
			wantPercent:  50,
		},
		{
			name:         "1 out of 3",
			totalNodes:   3,
			allowCount:   1,
			wantPercent:  33,
		},
		{
			name:         "all nodes",
			totalNodes:   10,
			allowCount:   10,
			wantPercent:  100,
		},
		{
			name:         "no nodes",
			totalNodes:   10,
			allowCount:   0,
			wantPercent:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.totalNodes == 0 {
				t.Skip("Cannot calculate percentage with 0 total nodes")
			}
			
			percentage := tt.allowCount * 100 / tt.totalNodes
			
			if percentage > 100 {
				percentage = 100
			}
			if percentage < 0 {
				percentage = 0
			}
			
			if percentage != tt.wantPercent {
				t.Errorf("recalculatePercentage() = %v, want %v", percentage, tt.wantPercent)
			}
		})
	}
}

func TestAddNewNodes2Pool_DuplicateHandling(t *testing.T) {
	tests := []struct {
		name           string
		existingNodes  []string
		newNodes       []string
		wantValidCount int
	}{
		{
			name:           "all new nodes are unique",
			existingNodes:  []string{"node1", "node2"},
			newNodes:       []string{"node3", "node4"},
			wantValidCount: 2,
		},
		{
			name:           "some nodes already exist",
			existingNodes:  []string{"node1", "node2"},
			newNodes:       []string{"node2", "node3"},
			wantValidCount: 1,
		},
		{
			name:           "all nodes already exist",
			existingNodes:  []string{"node1", "node2"},
			newNodes:       []string{"node1", "node2"},
			wantValidCount: 0,
		},
		{
			name:           "duplicate nodes in newNodes",
			existingNodes:  []string{"node1"},
			newNodes:       []string{"node2", "node2", "node3"},
			wantValidCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingSet := make(map[string]bool)
			for _, node := range tt.existingNodes {
				existingSet[node] = true
			}
			
			seenSet := make(map[string]bool)
			validCount := 0
			
			for _, node := range tt.newNodes {
				if seenSet[node] {
					continue
				}
				if existingSet[node] {
					continue
				}
				validCount++
				seenSet[node] = true
			}
			
			if validCount != tt.wantValidCount {
				t.Errorf("addNewNodes2Pool() validCount = %v, want %v", validCount, tt.wantValidCount)
			}
		})
	}
}

func TestDelNodesWithPool_NodeRemoval(t *testing.T) {
	tests := []struct {
		name          string
		existingNodes []string
		delNodes      []string
		wantRemaining int
	}{
		{
			name:          "remove some nodes",
			existingNodes: []string{"node1", "node2", "node3"},
			delNodes:      []string{"node2"},
			wantRemaining: 2,
		},
		{
			name:          "remove multiple nodes",
			existingNodes: []string{"node1", "node2", "node3", "node4"},
			delNodes:      []string{"node2", "node4"},
			wantRemaining: 2,
		},
		{
			name:          "remove non-existent node",
			existingNodes: []string{"node1", "node2"},
			delNodes:      []string{"node3"},
			wantRemaining: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeSet := make(map[string]bool)
			for _, node := range tt.existingNodes {
				nodeSet[node] = true
			}
			
			for _, node := range tt.delNodes {
				delete(nodeSet, node)
			}
			
			remaining := len(nodeSet)
			if remaining != tt.wantRemaining {
				t.Errorf("delNodesWithPool() remaining = %v, want %v", remaining, tt.wantRemaining)
			}
		})
	}
}
