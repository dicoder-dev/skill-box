#!/bin/bash

# 获取项目根目录
PROJECT_ROOT=$(cd "$(dirname "$0")/../../" && pwd)
GENCODE_DIR=$(cd "$(dirname "$0")" && pwd)
BIN_DIR="$GENCODE_DIR/bin"

# 获取GOPATH环境变量
if [ -z "$GOPATH" ]; then
    # 如果GOPATH未设置，尝试通过go env获取
    GOPATH=$(go env GOPATH)
    if [ -z "$GOPATH" ]; then
        echo "警告: 无法获取GOPATH环境变量，将只安装到本地bin目录"
    fi
fi

# 设置GOPATH/bin目录
if [ -n "$GOPATH" ]; then
    GOPATH_BIN="$GOPATH/bin"
    echo "GOPATH bin目录: $GOPATH_BIN"
fi

# 输出当前目录和项目根目录，用于调试
echo "当前目录: $(pwd)"
echo "项目根目录: $PROJECT_ROOT"
echo "Gencode目录: $GENCODE_DIR"
echo "Bin目录: $BIN_DIR"

# 确保bin目录存在
mkdir -p "$BIN_DIR"

# 编译gapi命令行工具
cd "$PROJECT_ROOT" && go build -o "$BIN_DIR/gapi" "$PROJECT_ROOT/cmd/gencode/main.go"

# 检查编译是否成功
if [ $? -eq 0 ]; then
    echo "GAPI 命令行工具编译成功！"
    echo "可执行文件位置: $BIN_DIR/gapi"
    
    # 使文件可执行
    chmod +x "$BIN_DIR/gapi"
    
    # 如果GOPATH存在，复制到GOPATH/bin目录
if [ -n "$GOPATH" ] && [ -d "$GOPATH" ]; then
    # 确保GOPATH/bin目录存在
    mkdir -p "$GOPATH_BIN"
    
    # 复制可执行文件到GOPATH/bin
    cp "$BIN_DIR/gapi" "$GOPATH_BIN/"
    chmod +x "$GOPATH_BIN/gapi"
    echo "已将gapi复制到GOPATH/bin目录: $GOPATH_BIN/gapi"
    echo "由于GOPATH/bin通常已在PATH中，您现在可以在任何地方使用gapi命令"
    echo "如果您无法直接使用gapi命令，请确保GOPATH/bin目录已添加到PATH环境变量中"
    echo "例如，在~/.bashrc或~/.zshrc中添加："
    echo "export PATH=\"$GOPATH_BIN:\$PATH\""
    
    # 提示用户刷新环境变量
    echo "\n您可以运行以下命令刷新当前终端的环境变量："
    echo "export PATH=\"$GOPATH_BIN:\$PATH\""
else
    echo "建议将此路径添加到您的PATH环境变量中，以便在任何地方使用gapi命令。"
    echo "例如，在~/.bashrc或~/.zshrc中添加："
    echo "export PATH=\"$BIN_DIR:\$PATH\""
fi
    
    # 创建符号链接到/usr/local/bin的选项（需要管理员权限）
    echo "\n如果您想在系统范围内使用gapi命令，可以运行以下命令（需要管理员权限）："
    echo "sudo ln -sf $BIN_DIR/gapi /usr/local/bin/gapi"
else
    echo "编译失败，请检查错误信息。"
    exit 1
fi