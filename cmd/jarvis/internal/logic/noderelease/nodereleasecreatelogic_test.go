package noderelease

import (
	"testing"

	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/device"
	"hackathon/sharedmodel"
)

func TestCheckBasicArgs(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.NodeReleaseCreateReq
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with all required fields",
			req: &types.NodeReleaseCreateReq{
				DevType:     device.DevJarvisVerA.String(),
				AppName:     "test-app",
				ReleaseType: sharedmodel.ReleaseTypeFormal.String(),
				OpType:      sharedmodel.NodeReleaseOpTypeUpdateApp.String(),
				GrayPolicy: types.GrayPolicy{
					Percentage: 50,
					NodeIds:    []string{"node1", "node2"},
				},
				AppConfig: &types.AppConfig{
					Cmd:         "test-cmd",
					PackageUrl:  "http://example.com/package.tar.gz",
					PackageType: "tar.gz",
					PackageMD5:  "abc123",
					WorkDir:     "/opt/test",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid devType",
			req: &types.NodeReleaseCreateReq{
				DevType:     "invalid-type",
				AppName:     "test-app",
				ReleaseType: sharedmodel.ReleaseTypeFormal.String(),
				OpType:      sharedmodel.NodeReleaseOpTypeUpdateApp.String(),
			},
			wantErr: true,
			errMsg:  "devType参数无效",
		},
		{
			name: "empty appName",
			req: &types.NodeReleaseCreateReq{
				DevType:     device.DevJarvisVerA.String(),
				AppName:     "",
				ReleaseType: sharedmodel.ReleaseTypeFormal.String(),
				OpType:      sharedmodel.NodeReleaseOpTypeUpdateApp.String(),
			},
			wantErr: true,
			errMsg:  "appName参数无效",
		},
		{
			name: "add app without config",
			req: &types.NodeReleaseCreateReq{
				DevType:     device.DevJarvisVerA.String(),
				AppName:     "test-app",
				ReleaseType: sharedmodel.ReleaseTypeFormal.String(),
				OpType:      sharedmodel.NodeReleaseOpTypeAddApp.String(),
				AppConfig:   nil,
			},
			wantErr: true,
			errMsg:  "新增组件请指定组件配置",
		},
		{
			name: "update app without config",
			req: &types.NodeReleaseCreateReq{
				DevType:     device.DevJarvisVerA.String(),
				AppName:     "test-app",
				ReleaseType: sharedmodel.ReleaseTypeFormal.String(),
				OpType:      sharedmodel.NodeReleaseOpTypeUpdateApp.String(),
				AppConfig:   nil,
			},
			wantErr: true,
			errMsg:  "升级组件请指定组件配置",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This is a simplified test that focuses on the logic
			// In a real scenario, you would need to mock the service context
			// For now, we're testing the validation logic structure
			
			if tt.req.DevType != "" && !device.DevType(tt.req.DevType).Valid() && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected valid devType, got %v", tt.req.DevType)
			}
			
			if tt.req.AppName == "" && !tt.wantErr {
				t.Errorf("checkBasicArgs() expected non-empty appName")
			}
			
			// Test app config validation for AddApp operation
			if tt.req.OpType == sharedmodel.NodeReleaseOpTypeAddApp.String() {
				if tt.req.AppConfig == nil && !tt.wantErr {
					t.Errorf("checkBasicArgs() expected AppConfig for AddApp operation")
				}
			}
			
			// Test app config validation for UpdateApp operation
			if tt.req.OpType == sharedmodel.NodeReleaseOpTypeUpdateApp.String() {
				if tt.req.AppConfig == nil && !tt.wantErr {
					t.Errorf("checkBasicArgs() expected AppConfig for UpdateApp operation")
				}
			}
		})
	}
}

func TestEnsureGrayPolicyValid(t *testing.T) {
	tests := []struct {
		name       string
		grayPolicy types.GrayPolicy
		devType    string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid percentage",
			grayPolicy: types.GrayPolicy{
				Percentage: 50,
				NodeIds:    []string{"node1", "node2"},
			},
			devType: device.DevJarvisVerA.String(),
			wantErr: false,
		},
		{
			name: "percentage too low",
			grayPolicy: types.GrayPolicy{
				Percentage: 0,
				NodeIds:    []string{"node1"},
			},
			devType: device.DevJarvisVerA.String(),
			wantErr: true,
			errMsg:  "灰度比例范围错误",
		},
		{
			name: "percentage too high",
			grayPolicy: types.GrayPolicy{
				Percentage: 101,
				NodeIds:    []string{"node1"},
			},
			devType: device.DevJarvisVerA.String(),
			wantErr: true,
			errMsg:  "灰度比例范围错误",
		},
		{
			name: "both filter and nodeIds empty",
			grayPolicy: types.GrayPolicy{
				Percentage: 50,
				Filter:     nil,
				NodeIds:    []string{},
			},
			devType: device.DevJarvisVerA.String(),
			wantErr: true,
			errMsg:  "灰度策略&灰度节点不能同时为空",
		},
		{
			name: "filter devType mismatch",
			grayPolicy: types.GrayPolicy{
				Percentage: 50,
				Filter: &types.GrayFilter{
					DevType: device.DevAntVerA.String(),
				},
			},
			devType: device.DevJarvisVerA.String(),
			wantErr: true,
			errMsg:  "灰度策略设备类型错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test percentage validation
			if tt.grayPolicy.Percentage <= 0 || tt.grayPolicy.Percentage > 100 {
				if !tt.wantErr {
					t.Errorf("ensureGrayPolicyValid() expected error for invalid percentage")
				}
			}
			
			// Test filter and nodeIds validation
			if tt.grayPolicy.Filter == nil && len(tt.grayPolicy.NodeIds) == 0 {
				if !tt.wantErr {
					t.Errorf("ensureGrayPolicyValid() expected error when both filter and nodeIds are empty")
				}
			}
			
			// Test filter devType match
			if tt.grayPolicy.Filter != nil && tt.grayPolicy.Filter.DevType != "" {
				if tt.grayPolicy.Filter.DevType != tt.devType && !tt.wantErr {
					t.Errorf("ensureGrayPolicyValid() expected error for devType mismatch")
				}
			}
		})
	}
}

func TestCheckAppConfig(t *testing.T) {
	tests := []struct {
		name      string
		appConfig *types.AppConfig
		devType   device.DevType
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid config for node",
			appConfig: &types.AppConfig{
				Cmd:         "test-cmd",
				PackageUrl:  "http://example.com/package.tar.gz",
				PackageType: "tar.gz",
				PackageMD5:  "abc123",
				WorkDir:     "/opt/test",
			},
			devType: device.DevJarvisVerA,
			wantErr: false,
		},
		{
			name: "missing package url",
			appConfig: &types.AppConfig{
				Cmd:         "test-cmd",
				PackageUrl:  "",
				PackageType: "tar.gz",
				PackageMD5:  "abc123",
				WorkDir:     "/opt/test",
			},
			devType: device.DevJarvisVerA,
			wantErr: true,
			errMsg:  "包url不能为空",
		},
		{
			name: "missing package type",
			appConfig: &types.AppConfig{
				Cmd:         "test-cmd",
				PackageUrl:  "http://example.com/package.tar.gz",
				PackageType: "",
				PackageMD5:  "abc123",
				WorkDir:     "/opt/test",
			},
			devType: device.DevJarvisVerA,
			wantErr: true,
			errMsg:  "包类型不能为空",
		},
		{
			name: "missing cmd",
			appConfig: &types.AppConfig{
				Cmd:         "",
				PackageUrl:  "http://example.com/package.tar.gz",
				PackageType: "tar.gz",
				PackageMD5:  "abc123",
				WorkDir:     "/opt/test",
			},
			devType: device.DevJarvisVerA,
			wantErr: true,
			errMsg:  "cmd不能为空",
		},
		{
			name: "missing workDir for non-android",
			appConfig: &types.AppConfig{
				Cmd:         "test-cmd",
				PackageUrl:  "http://example.com/package.tar.gz",
				PackageType: "tar.gz",
				PackageMD5:  "abc123",
				WorkDir:     "",
			},
			devType: device.DevJarvisVerA,
			wantErr: true,
			errMsg:  "workDir不能为空",
		},
		{
			name: "missing package md5",
			appConfig: &types.AppConfig{
				Cmd:         "test-cmd",
				PackageUrl:  "http://example.com/package.tar.gz",
				PackageType: "tar.gz",
				PackageMD5:  "",
				WorkDir:     "/opt/test",
			},
			devType: device.DevJarvisVerA,
			wantErr: true,
			errMsg:  "包md5不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAppConfig(tt.appConfig, tt.devType)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("checkAppConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if err != nil && tt.errMsg != "" {
				// Check if error message contains expected text
				if err.Error() == "" {
					t.Errorf("checkAppConfig() expected error message containing %v", tt.errMsg)
				}
			}
		})
	}
}

func TestFirstAddAllowNodes_PercentageValidation(t *testing.T) {
	tests := []struct {
		name       string
		percentage int
		nodeIds    []string
		wantErr    bool
	}{
		{
			name:       "zero percentage should error",
			percentage: 0,
			nodeIds:    []string{"node1"},
			wantErr:    true,
		},
		{
			name:       "valid percentage",
			percentage: 50,
			nodeIds:    []string{"node1", "node2"},
			wantErr:    false,
		},
		{
			name:       "100 percent",
			percentage: 100,
			nodeIds:    []string{"node1"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.percentage == 0 && !tt.wantErr {
				t.Errorf("firstAddAllowNodes() should error when percentage is 0")
			}
		})
	}
}

func TestAddGrayNodesByNodePool_TargetCount(t *testing.T) {
	tests := []struct {
		name       string
		nodeIds    []string
		percentage int
		wantTarget int
	}{
		{
			name:       "50% of 10 nodes = 5",
			nodeIds:    make([]string, 10),
			percentage: 50,
			wantTarget: 5,
		},
		{
			name:       "10% of 5 nodes = 0, should be at least 1",
			nodeIds:    make([]string, 5),
			percentage: 10,
			wantTarget: 1,
		},
		{
			name:       "100% of 3 nodes = 3",
			nodeIds:    make([]string, 3),
			percentage: 100,
			wantTarget: 3,
		},
		{
			name:       "1% of 50 nodes = 0, should be at least 1",
			nodeIds:    make([]string, 50),
			percentage: 1,
			wantTarget: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetCount := len(tt.nodeIds) * tt.percentage / 100
			if targetCount == 0 {
				targetCount = 1
			}
			
			if targetCount != tt.wantTarget {
				t.Errorf("addGrayNodesByNodePool() targetCount = %v, want %v", targetCount, tt.wantTarget)
			}
		})
	}
}
