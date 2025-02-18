package envmanager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	envLoaderName   = "aspnmy_envloader"
	installedPath   = "/usr/local/bin/aspnmy_envloader"
	localPath       = "env_loader/aspnmy_envloader"
	testEnvKey      = "testenv"
	testEnvValue    = "test_value_123"
	envLoaderDirKey = "aspnmy_envloaderDir" // 新增：组件路径环境变量名
)

// verifyEnvLoaderComponent 验证组件是否可用
func verifyEnvLoaderComponent(path string) error {
	cmd := exec.Command(path, "ver")
	output, err := cmd.Output()
	if err != nil {

		return fmt.Errorf("组件验证失败: %v", err)
	}
	if len(output) == 0 {
		return fmt.Errorf("组件验证失败: 未返回版本信息")
	}
	return nil
}

// verifyEnvLoaderComponent 验证组件是否可用

func verifyEnvLoaderName(envLoaderName string) error {
	cmd := exec.Command(envLoaderName, "ver")
	output, err := cmd.Output()
	if err != nil {

		return fmt.Errorf("组件验证失败: %v", err)
	}
	if len(output) == 0 {
		return fmt.Errorf("组件验证失败: 未返回版本信息")
	}
	return nil
}

// verifyAndGetCommand 获取组件并验证可用性
func verifyAndGetCommand(path string) (string, error) {
	// 1. 首先尝试完整路径验证
	if err := verifyEnvLoaderComponent(path); err == nil {
		return path, nil
	}

	// 2. 如果完整路径验证失败，尝试仅组件名验证
	if err := verifyEnvLoaderName(envLoaderName); err == nil {
		return envLoaderName, nil
	}

	return "", fmt.Errorf("组件验证失败: 路径验证和名称验证均失败")
}

// updateSystemPath 将组件路径添加到系统PATH
func updateSystemPath(componentDir string) error {
	currentPath := os.Getenv("PATH")
	if !strings.Contains(currentPath, componentDir) {
		newPath := fmt.Sprintf("%s:%s", currentPath, componentDir)
		if err := os.Setenv("PATH", newPath); err != nil {
			return fmt.Errorf("更新PATH环境变量失败: %v", err)
		}
	}
	return nil
}

// findEnvLoader 查找 aspnmy_envloader 组件
func findEnvLoader() (string, error) {
	// 优先检查命令是否可以直接使用
	if err := verifyEnvLoaderName(envLoaderName); err == nil {
		// 即使可以直接使用命令，也应该返回实际路径
		if path, err := exec.LookPath(envLoaderName); err == nil {
			return path, nil
		}
	}

	// 获取当前包所在目录
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("无法获取当前文件路径")
	}

	// 修改：直接从当前包目录查找组件
	pkgDir := filepath.Dir(filename)
	componentPath := filepath.Join(pkgDir, localPath)

	// 检查组件是否存在并可用
	if _, err := os.Stat(componentPath); err == nil {
		if err := verifyEnvLoaderComponent(componentPath); err == nil {
			// 更新系统环境变量
			if err := updateSystemPath(filepath.Dir(componentPath)); err == nil {
				if err := UpdateEnvironmentVariable(envLoaderDirKey, filepath.Dir(componentPath)); err == nil {
					return componentPath, nil
				}
			}
		}
	}

	// 如果找不到，尝试更新
	log.Printf("未找到可用组件，正在尝试下载最新版本...")
	updateScript := filepath.Join(pkgDir, "env_loader/update.sh")
	if _, err := os.Stat(updateScript); err == nil {
		cmd := exec.Command("bash", updateScript)
		if err := cmd.Run(); err == nil {
			// 更新后重新检查组件
			if _, err := os.Stat(componentPath); err == nil {
				if err := verifyEnvLoaderComponent(componentPath); err == nil {
					return componentPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("未找到可用的 aspnmy_envloader 组件，且自动更新失败")
}

// ExecEnvLoader 执行环境变量加载器命令
func ExecEnvLoader(command string) error {
	loaderPath, err := findEnvLoader()
	if err != nil {
		return fmt.Errorf("查找 aspnmy_envloader 失败: %v", err)
	}

	// 验证组件并获取正确的执行命令
	cmdPath, err := verifyAndGetCommand(loaderPath)
	if err != nil {
		return fmt.Errorf("验证组件失败: %v", err)
	}

	// 使用获取到的正确命令执行
	cmd := exec.Command(cmdPath, command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 %s %s 失败: %w", cmdPath, command, err)
	}
	return nil
}

// ReloadEnv 重新加载环境变量
func ReloadEnv() error {
	// 先执行 reload 加载配置文件
	if err := ExecEnvLoader("reload"); err != nil {
		return fmt.Errorf("加载环境变量失败: %v", err)
	}

	// 再 source ~/.bashrc 使环境变量生效
	cmd := exec.Command("bash", "-c", "source ~/.bashrc")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 source ~/.bashrc 失败: %v", err)
	}

	return nil
}

// UpdateEnvironmentVariable 更新环境变量
func UpdateEnvironmentVariable(key, value string) error {
	// 1. 获取 .env 文件路径
	BaseDir := os.Getenv("ollama_scannerBaseDir")
	envFile := filepath.Join(BaseDir, ".env")

	// 2. 读取当前内容
	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("读取 .env 文件失败: %v", err)
	}

	// 3. 按行分割
	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))
	found := false

	// 4. 查找并更新或保留现有行
	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			found = true
			continue
		}
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	// 5. 添加新的环境变量
	newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))

	// 6. 写回文件
	if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %v", err)
	}

	// 7. 重新加载环境变量
	if err := ReloadEnv(); err != nil {
		return fmt.Errorf("重新加载环境变量失败: %v", err)
	}

	// 8. 验证更新结果
	newValue := os.Getenv(key)
	if newValue != value {
		return fmt.Errorf("环境变量更新失败：%s 值不匹配", key)
	}

	if !found {
		log.Printf("新增环境变量: %s=%s", key, value)
	} else {
		log.Printf("更新环境变量: %s=%s", key, value)
	}

	return nil
}

// VerifyEnvLoader 验证环境变量加载器是否正常工作
func VerifyEnvLoader() error {
	// 写入测试变量
	if err := UpdateEnvironmentVariable(testEnvKey, testEnvValue); err != nil {
		return fmt.Errorf("写入测试环境变量失败: %v", err)
	}

	// 验证是否能正确读取
	value := os.Getenv(testEnvKey)
	if value != testEnvValue {
		return fmt.Errorf("验证失败: 期望值 %s, 实际值 %s", testEnvValue, value)
	}

	// 清理测试变量
	if err := UpdateEnvironmentVariable(testEnvKey, ""); err != nil {
		return fmt.Errorf("清理测试环境变量失败: %v", err)
	}

	return nil
}

// RemoveEnvironmentVariable 删除环境变量
func RemoveEnvironmentVariable(key string) error {
	// 验证环境变量加载器
	if err := VerifyEnvLoader(); err != nil {
		return fmt.Errorf("环境变量加载器验证失败: %v", err)
	}

	// 1. 获取 .env 文件路径
	BaseDir := os.Getenv("ollama_scannerBaseDir")
	envFile := filepath.Join(BaseDir, ".env")

	// 2. 读取当前内容
	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("读取 .env 文件失败: %v", err)
	}

	// 3. 按行分割并过滤要删除的变量
	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))
	found := false

	for _, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			found = true
			continue
		}
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	if !found {
		log.Printf("警告: 未找到要删除的环境变量: %s", key)
		return nil
	}

	// 4. 写回文件
	if err := os.WriteFile(envFile, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %v", err)
	}

	// 5. 重新加载环境变量
	if err := ReloadEnv(); err != nil {
		return fmt.Errorf("重新加载环境变量失败: %v", err)
	}

	// 6. 验证删除结果
	if value := os.Getenv(key); value != "" {
		return fmt.Errorf("环境变量删除失败：%s 仍然存在，值为 %s", key, value)
	}

	log.Printf("已删除环境变量: %s", key)
	return nil
}
