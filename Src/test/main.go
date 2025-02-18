package main

import (
	"fmt"
	"log"

	"github.com/aspnmy/ollama_scanner/pkg/envmanager"
)

func main() {
	// 测试环境变量加载器
	if err := envmanager.VerifyEnvLoader(); err != nil {
		log.Fatalf("环境变量加载器验证失败: %v", err)
	}
	fmt.Println("✅ 环境变量加载器验证成功")

	// 测试环境变量的增删改
	testKey := "TEST_VAR"
	testValue := "test_value"

	// 测试添加
	if err := envmanager.UpdateEnvironmentVariable(testKey, testValue); err != nil {
		log.Fatalf("添加环境变量失败: %v", err)
	}
	fmt.Printf("✅ 成功添加环境变量 %s=%s\n", testKey, testValue)

	// 测试删除
	if err := envmanager.RemoveEnvironmentVariable(testKey); err != nil {
		log.Fatalf("删除环境变量失败: %v", err)
	}
	fmt.Printf("✅ 成功删除环境变量 %s\n", testKey)

	fmt.Println("✅ 所有测试通过")
}
