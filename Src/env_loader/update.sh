#!/bin/bash

# 修改：使用脚本所在目录作为安装目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$SCRIPT_DIR"
REPO_URL="https://github.com/aspnmy/ollama_scanner_load_env"
RELEASE_API="https://api.github.com/repos/aspnmy/ollama_scanner_load_env/releases"

# 打印带颜色的消息
print_message() {
    local color=$1
    local message=$2
    echo -e "\033[${color}m${message}\033[0m"
}

# 检查必要的命令
check_dependencies() {
    local deps=("curl" "jq" "tar")
    for cmd in "${deps[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            print_message "31" "错误: 未找到必要的命令: $cmd"
            exit 1
        fi
    done
}

# 获取最新版本号
get_latest_version() {
    curl -s "$RELEASE_API/latest" | jq -r '.tag_name'
}

# 获取并验证SHA256
get_and_verify_sha256() {
    local version=$1
    local file=$2
    local sha256_url

    print_message "34" "正在获取SHA256校验值..."
    
    # 获取SHA256校验值URL
    if [[ "$version" == "latest" ]]; then
        sha256_url=$(curl -s "$RELEASE_API/latest" | jq -r '.assets[] | select(.name | endswith(".sha256")) | .browser_download_url')
    else
        sha256_url=$(curl -s "$RELEASE_API/tags/$version" | jq -r '.assets[] | select(.name | endswith(".sha256")) | .browser_download_url')
    fi

    if [[ -z "$sha256_url" || "$sha256_url" == "null" ]]; then
        print_message "31" "错误: 未找到SHA256校验文件"
        return 1
    fi

    # 下载SHA256校验值
    local expected_sha256=$(curl -sL "$sha256_url" | cut -d' ' -f1)
    if [[ -z "$expected_sha256" ]]; then
        print_message "31" "错误: 无法获取SHA256校验值"
        return 1
    fi

    # 计算下载文件的SHA256
    local actual_sha256=$(sha256sum "$file" | cut -d' ' -f1)

    # 验证SHA256
    if [[ "$expected_sha256" != "$actual_sha256" ]]; then
        print_message "31" "安全性错误: SHA256校验失败"
        print_message "31" "期望的SHA256: $expected_sha256"
        print_message "31" "实际的SHA256: $actual_sha256"
        return 1
    fi

    print_message "32" "SHA256校验通过"
    return 0
}

# 备份现有组件
backup_component() {
    local component="$1"
    if [[ -f "$component" ]]; then
        # 使用上海时区的时间戳
        local timestamp=$(TZ='Asia/Shanghai' date '+%Y%m%d_%H%M%S')
        local backup_file="${component}.${timestamp}.bak"
        
        print_message "34" "正在备份现有组件到: $backup_file"
        if ! cp "$component" "$backup_file"; then
            print_message "31" "错误: 备份失败"
            return 1
        fi
        print_message "32" "备份完成"
    fi
    return 0
}

# 下载并安装指定版本
install_version() {
    local version=$1
    local temp_dir="$INSTALL_DIR/.temp"
    local asset_url

    # 确保临时目录存在且为空
    rm -rf "$temp_dir"
    mkdir -p "$temp_dir"

    print_message "34" "正在获取版本 $version 的下载信息..."
    
    # 获取下载URL
    if [[ "$version" == "latest" ]]; then
        asset_url=$(curl -s "$RELEASE_API/latest" | jq -r '.assets[] | select(.name | endswith(".tar.gz")) | .browser_download_url')
    else
        asset_url=$(curl -s "$RELEASE_API/tags/$version" | jq -r '.assets[] | select(.name | endswith(".tar.gz")) | .browser_download_url')
    fi

    if [[ -z "$asset_url" || "$asset_url" == "null" ]]; then
        print_message "31" "错误: 未找到版本 $version 的下载链接"
        rm -rf "$temp_dir"
        exit 1
    fi

    # 下载文件前先清理并创建目录
    print_message "34" "正在下载组件..."
    if ! curl -L "$asset_url" -o "$temp_dir/release.tar.gz"; then
        print_message "31" "错误: 下载失败"
        rm -rf "$temp_dir"
        exit 1
    fi

    # SHA256校验
    if ! get_and_verify_sha256 "$version" "$temp_dir/release.tar.gz"; then
        rm -rf "$temp_dir"
        exit 1
    fi

    # 解压文件
    print_message "34" "正在解压组件..."
    if ! tar -xzf "$temp_dir/release.tar.gz" -C "$temp_dir"; then
        print_message "31" "错误: 解压失败"
        rm -rf "$temp_dir"
        exit 1
    fi

    # 修改：递归查找组件文件
    print_message "34" "正在查找组件..."
    local component_path=$(find "$temp_dir" -type f -name "aspnmy_envloader")
    
    if [[ -n "$component_path" ]]; then
        print_message "34" "找到组件: $component_path"
        
        # 添加备份步骤
        if ! backup_component "$INSTALL_DIR/aspnmy_envloader"; then
            print_message "31" "错误: 备份失败，取消安装"
            rm -rf "$temp_dir"
            exit 1
        fi

        
        # 首先安装到系统中保证组件函数能够正常运行
        cp "$component_path" "/usr/local/bin/aspnmy_envloader"
        chmod +x /usr/local/bin/aspnmy_envloader
        # 然后再安装到脚本目录中
        mv "$component_path" "$INSTALL_DIR/aspnmy_envloader"
        chmod +x "$INSTALL_DIR/aspnmy_envloader"
        print_message "32" "安装成功!"
    else
        print_message "31" "错误: 未找到组件文件，解压内容如下:"
        ls -R "$temp_dir"
        rm -rf "$temp_dir"
        exit 1
    fi

    # 清理临时文件
    rm -rf "$temp_dir"
}

# 主函数
main() {
    check_dependencies

    local version="$1"
    if [[ -z "$version" ]]; then
        version="latest"
    fi

    # 检查当前版本
    if [[ -f "$INSTALL_DIR/aspnmy_envloader" ]]; then
        current_version=$("$INSTALL_DIR/aspnmy_envloader" ver | grep -oE "Version:[0-9.]+")
        print_message "34" "当前版本: $current_version"
    fi

    # 安装指定版本
    install_version "$version"
}

# 脚本使用说明
usage() {
    echo "用法: $0 [版本号]"
    echo "示例:"
    echo "  $0 v1.0.1   # 安装指定版本"
}

# 参数检查
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    usage
    exit 0
fi

main "$@"
