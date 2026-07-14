#requires -version 5.1
<#
.SYNOPSIS
    Windows Skill Box 升级替身脚本
.DESCRIPTION
    由 desktop/services/updater_helper.go fork;父进程 Quit 后接管二进制替换 + 重启。
.PARAMETER Dest
    已下载的 zip / installer 路径(Downloader.Download 返的本地路径)
.PARAMETER TargetInstallDir
    Skill Box 当前安装目录(与 wails app.exe 同目录,例如 C:\Program Files\SkillBox)
.PARAMETER OS
    平台标识(总是 "windows")
.PARAMETER Arch
    amd64 / arm64
.PARAMETER OldVersion
    升级前版本,作为 SKILLBOX_UPDATER_FROM 写到 env 给新进程
#>
param(
    [Parameter(Mandatory=$true)][string]$Dest,
    [Parameter(Mandatory=$true)][string]$TargetInstallDir,
    [string]$OS = "windows",
    [string]$Arch = "amd64",
    [string]$OldVersion = ""
)

$ErrorActionPreference = "Stop"

# 等父进程完全退出
Start-Sleep -Seconds 2

$targetExe = Join-Path $TargetInstallDir "skill-box.exe"
$bak = "$targetExe.bak"

try {
    # 备份当前二进制
    if (Test-Path $targetExe) {
        Copy-Item -Force $targetExe $bak
    }

    # 解压 zip 到 target 目录(覆盖 skill-box.exe + WebView2Loader.dll + .syso 等)
    if ($Dest -like "*.zip") {
        $expandDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString())) -Force
        Expand-Archive -Force -Path $Dest -DestinationPath $expandDir.FullName
        # 把内部 skill-box.exe 拷过去(以及可能附带的 webview2 dll)
        Copy-Item -Force (Join-Path $expandDir.FullName "skill-box.exe") $targetExe
        # 顺手同步 webview2 runtime dll(若有)
        $dll = Join-Path $expandDir.FullName "WebView2Loader.dll"
        if (Test-Path $dll) {
            Copy-Item -Force $dll (Join-Path $TargetInstallDir "WebView2Loader.dll")
        }
        Remove-Item -Recurse -Force $expandDir.FullName
    } else {
        # installer / msi 走 msiexec / passive(本期先 rollback,因为需要单独的 nsis diff 路径)
        throw "windows helper: unsupported dest extension (only .zip): $Dest"
    }

    # 启动新进程,带 SKILLBOX_UPDATER_FROM 让 desktop.startupAsync 弹"升级成功/失败"
    $env:SKILLBOX_UPDATER_FROM = $OldVersion
    Start-Process -FilePath $targetExe
    Remove-Item -Force $bak -ErrorAction SilentlyContinue
    exit 0
}
catch {
    # 失败回滚
    if (Test-Path $bak) {
        Copy-Item -Force $bak $targetExe
        Remove-Item -Force $bak -ErrorAction SilentlyContinue
    }
    Write-Error "helper_windows.ps1 failed: $_"
    exit 1
}
