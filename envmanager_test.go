package envmanager

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setupMockEnvLoader 创建模拟的环境变量加载器组件
func setupMockEnvLoader(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "mock_envloader")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建 env_loader 目录
	loaderDir := filepath.Join(tmpDir, "env_loader")
	if err := os.MkdirAll(loaderDir, 0755); err != nil {
		t.Fatalf("创建加载器目录失败: %v", err)
	}

	// 创建模拟的 aspnmy_envloader
	mockLoader := filepath.Join(loaderDir, "aspnmy_envloader")
	script := `#!/bin/bash
case "$1" in
  "ver")
    echo "1.0.0"
    ;;
  "reload")
    exit 0
    ;;
  *)
    exit 1
    ;;
esac`

	if err := os.WriteFile(mockLoader, []byte(script), 0755); err != nil {
		t.Fatalf("创建模拟加载器失败: %v", err)
	}

	// 设置环境变量
	os.Setenv(envLoaderDirKey, loaderDir)

	cleanup := func() {
		os.Unsetenv(envLoaderDirKey)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestVerifyEnvLoaderComponent(t *testing.T) {
	// Create temp test directory
	tmpDir, err := os.MkdirTemp("", "envloader_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		setup   func(dir string) string
		wantErr bool
	}{
		{
			name: "non-existent component",
			setup: func(dir string) string {
				return filepath.Join(dir, "non-existent")
			},
			wantErr: true,
		},
		{
			name: "component exists but returns no output",
			setup: func(dir string) string {
				path := filepath.Join(dir, "empty_output")
				script := `#!/bin/bash
exit 0`
				if err := os.WriteFile(path, []byte(script), 0755); err != nil {
					t.Fatalf("Failed to create test script: %v", err)
				}
				return path
			},
			wantErr: true,
		},
		{
			name: "valid component",
			setup: func(dir string) string {
				path := filepath.Join(dir, "valid_component")
				script := `#!/bin/bash
echo "1.0.0"`
				if err := os.WriteFile(path, []byte(script), 0755); err != nil {
					t.Fatalf("Failed to create test script: %v", err)
				}
				return path
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(tmpDir)
			err := verifyEnvLoaderComponent(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyEnvLoaderComponent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateSystemPath(t *testing.T) {
	tests := []struct {
		name         string
		initialPath  string
		componentDir string
		wantErr      bool
		wantPath     string
	}{
		{
			name:         "add new path",
			initialPath:  "/usr/local/bin:/usr/bin",
			componentDir: "/opt/myapp/bin",
			wantErr:      false,
			wantPath:     "/usr/local/bin:/usr/bin:/opt/myapp/bin",
		},
		{
			name:         "path already exists",
			initialPath:  "/usr/local/bin:/opt/myapp/bin:/usr/bin",
			componentDir: "/opt/myapp/bin",
			wantErr:      false,
			wantPath:     "/usr/local/bin:/opt/myapp/bin:/usr/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 备份当前 PATH
			oldPath := os.Getenv("PATH")
			defer os.Setenv("PATH", oldPath)

			// 设置测试环境
			os.Setenv("PATH", tt.initialPath)

			err := updateSystemPath(tt.componentDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateSystemPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotPath := os.Getenv("PATH")
			if !tt.wantErr && gotPath != tt.wantPath {
				t.Errorf("PATH = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestFindEnvLoader(t *testing.T) {
	// 1. 获取测试目录（当前包所在目录）
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法获取当前文件路径")
	}
	pkgDir := filepath.Dir(filename)
	projectRoot := filepath.Join(pkgDir, "../..")

	// 2. 创建真实的组件目录和文件
	loaderDir := filepath.Join(projectRoot, "env_loader")
	if err := os.MkdirAll(loaderDir, 0755); err != nil {
		t.Fatalf("创建组件目录失败: %v", err)
	}

	loaderPath := filepath.Join(loaderDir, "aspnmy_envloader")
	script := `#!/bin/bash
case "$1" in
  "ver")
    echo "1.0.0"
    ;;
  "reload")
    exit 0
    ;;
  *)
    exit 1
    ;;
esac`
	if err := os.WriteFile(loaderPath, []byte(script), 0755); err != nil {
		t.Fatalf("创建组件文件失败: %v", err)
	}

	// 3. 测试组件查找
	path, err := findEnvLoader()
	if err != nil {
		t.Fatalf("findEnvLoader() 失败: %v", err)
	}

	// 4. 验证找到的路径是否正确
	if path != loaderPath {
		t.Errorf("findEnvLoader() 返回路径不正确, got = %v, want = %v", path, loaderPath)
	}

	// 5. 验证环境变量是否正确设置
	if envDir := os.Getenv(envLoaderDirKey); envDir != filepath.Dir(path) {
		t.Errorf("组件目录环境变量设置不正确, got = %v, want = %v", envDir, filepath.Dir(path))
	}
}

// 在确认组件查找正确后，其他测试可以直接使用该组件
func TestUpdateEnvironmentVariable(t *testing.T) {
	// 1. 创建临时的.env文件
	tmpDir, err := os.MkdirTemp("", "env_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("EXISTING_VAR=old_value\n"), 0644); err != nil {
		t.Fatalf("创建测试环境文件失败: %v", err)
	}

	// 2. 设置必要的环境变量
	oldBaseDir := os.Getenv("ollama_scannerBaseDir")
	os.Setenv("ollama_scannerBaseDir", tmpDir)
	defer os.Setenv("ollama_scannerBaseDir", oldBaseDir)

	// 3. 运行测试用例
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "add new variable",
			key:     "NEW_VAR",
			value:   "new_value",
			wantErr: false,
		},
		{
			name:    "update existing variable",
			key:     "EXISTING_VAR",
			value:   "updated_value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateEnvironmentVariable(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateEnvironmentVariable() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 验证环境变量是否正确设置
				if got := os.Getenv(tt.key); got != tt.value {
					t.Errorf("环境变量值 = %v, want %v", got, tt.value)
				}
			}
		})
	}
}

func TestRemoveEnvironmentVariable(t *testing.T) {
	// 设置模拟组件
	tmpDir, cleanup := setupMockEnvLoader(t)
	defer cleanup()

	// 创建测试 .env 文件
	envFile := filepath.Join(tmpDir, ".env")
	initialEnv := "TEST_VAR=test_value\nKEEP_VAR=keep_value\n"
	if err := os.WriteFile(envFile, []byte(initialEnv), 0644); err != nil {
		t.Fatalf("创建测试环境文件失败: %v", err)
	}

	// 设置基础目录环境变量
	oldBaseDir := os.Getenv("ollama_scannerBaseDir")
	os.Setenv("ollama_scannerBaseDir", tmpDir)
	defer os.Setenv("ollama_scannerBaseDir", oldBaseDir)

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "remove existing variable",
			key:     "TEST_VAR",
			wantErr: false,
		},
		{
			name:    "remove non-existent variable",
			key:     "NON_EXISTENT_VAR",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RemoveEnvironmentVariable(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveEnvironmentVariable() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 验证环境变量是否被删除
				if got := os.Getenv(tt.key); got != "" {
					t.Errorf("环境变量 %s 未被正确删除, 值为 %v", tt.key, got)
				}
			}
		})
	}
}

func TestReloadEnv(t *testing.T) {
	// 设置模拟组件
	_, cleanup := setupMockEnvLoader(t)
	defer cleanup()

	// 备份原始环境变量
	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)

	err := ReloadEnv()
	if err != nil {
		t.Errorf("ReloadEnv() error = %v", err)
	}
}
