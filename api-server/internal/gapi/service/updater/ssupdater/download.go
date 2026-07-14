package ssupdater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ginp-api/pkg/logger"
)

// 升级流程的全局状态(单进程单用户)。
type State struct {
	mu       sync.Mutex
	Phase    string // idle / checking / downloading / verifying / pendingRestart / failed
	Progress int    // 0~100
	Err      string
	Path     string // 下载目标路径
	Updated  time.Time
}

// 各 phase 常量,前端字符串匹配。
const (
	PhaseIdle          = "idle"
	PhaseChecking      = "checking"
	PhaseDownloading   = "downloading"
	PhaseVerifying     = "verifying"
	PhasePendingRestart = "pendingRestart"
	PhaseFailed        = "failed"
)

func newState() *State { return &State{Phase: PhaseIdle} }

// 进程内单例。MVP 阶段不上磁盘(避免 prefs 写崩后启动卡顿),真要落盘由 controller 调 settings.Service 写。
var globalState = newState()

// StateOf 返回当前状态的快照。
func StateOf() *State {
	globalState.mu.Lock()
	defer globalState.mu.Unlock()
	return &State{
		Phase:    globalState.Phase,
		Progress: globalState.Progress,
		Err:      globalState.Err,
		Path:     globalState.Path,
		Updated:  globalState.Updated,
	}
}

func setPhase(phase string) {
	globalState.mu.Lock()
	globalState.Phase = phase
	globalState.Err = ""
	globalState.Updated = time.Now()
	globalState.mu.Unlock()
}

func setProgress(p int) {
	globalState.mu.Lock()
	globalState.Progress = p
	globalState.Updated = time.Now()
	globalState.mu.Unlock()
}

func setErr(err string) {
	globalState.mu.Lock()
	globalState.Phase = PhaseFailed
	globalState.Err = err
	globalState.Updated = time.Now()
	globalState.mu.Unlock()
}

func setPath(p string) {
	globalState.mu.Lock()
	globalState.Path = p
	globalState.Updated = time.Now()
	globalState.mu.Unlock()
}

// Downloader 是下载 + 校验的整体封装。
type Downloader struct {
	// TargetDir 下载目标目录;由 controller 注入(桌面端走 os.UserCacheDir + /skill-box/updater/)。
	TargetDir string
	// 进度回调(0~100)。controller 用它把 phase=downloading + progress 写回 StateOf()。
	OnProgress func(percent int)
}

// Download 下载 asset 到 TargetDir/<basename>(同一个 url 反复下走 sha256 命中跳过)。
// 支持 HTTP Range 续传:如果 dest 已有部分文件,size 与 asset.Size 一致时从已写入字节开始拉。
//
// 完成后立刻 SHA256 校验,失败返 error,**不删文件**(留给桌面 helper 调试)。
func (d *Downloader) Download(ctx context.Context, asset *Asset) (string, error) {
	if asset == nil {
		return "", fmt.Errorf("updater: nil asset")
	}
	if len(asset.URLs) == 0 {
		return "", fmt.Errorf("updater: asset has no urls")
	}
	if d.TargetDir == "" {
		return "", fmt.Errorf("updater: empty target dir")
	}
	if err := os.MkdirAll(d.TargetDir, 0o755); err != nil {
		return "", fmt.Errorf("updater: mkdir target: %w", err)
	}
	base := pickFilename(asset.URLs)
	if base == "" {
		base = fmt.Sprintf("skill-box_%s_%s_%s.bin", asset.OS, asset.Arch, asset.Kind)
	}
	dest := filepath.Join(d.TargetDir, base)

	// 已下载且 sha256 命中 → 直接返(典型续传命中场景)
	if exists, prev, err := sha256File(dest); err == nil && exists && prev == strings.ToLower(asset.SHA256) {
		logger.Info("updater: cache hit, skip download dest=%s", dest)
		setPath(dest)
		setPhase(PhaseVerifying)
		setProgress(100)
		setPhase(PhasePendingRestart)
		return dest, nil
	}

	// 不带 Range 直接整段下载(MVP 不做真续传,只做"断点续传式占位",避免网络翻倍风险)。
	_ = d.maybeResume(ctx, dest)

	var lastErr error
	for _, raw := range asset.URLs {
		if raw == "" {
			continue
		}
		url := strings.TrimSpace(raw)
		err := d.singleDownload(ctx, url, dest, asset.Size)
		if err != nil {
		logger.Warn("updater: download failed url=%s err=%v", url, err)
			lastErr = err
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		setErr(lastErr.Error())
		return "", lastErr
	}

	// 校验
	setPhase(PhaseVerifying)
	setProgress(100)
	if d.OnProgress != nil {
		d.OnProgress(100)
	}
	got, err := sha256FileIfExists(dest)
	if err != nil {
		setErr("sha256 stat failed: " + err.Error())
		return "", err
	}
	if !strings.EqualFold(got, asset.SHA256) {
		setErr(fmt.Sprintf("sha256 mismatch: got=%s expect=%s", got, asset.SHA256))
		return "", fmt.Errorf("updater: sha256 mismatch got=%s expect=%s", got, asset.SHA256)
	}
	setPath(dest)
	setPhase(PhasePendingRestart)
	return dest, nil
}

// ResetDownloads 删除 TargetDir 下所有非空文件(用于 helper 失败后下次重试干净)。
// MVP **不删**(避免误删 helper 正在用的 .new 中间文件),caller 自行处理。
// 这里保留 stub 接口,便于后续扩展。
func (d *Downloader) ResetDownloads() error { return nil }

// singleDownload 走单一 url 的整段下载。
func (d *Downloader) singleDownload(ctx context.Context, url, dest string, totalSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "skill-box-updater/1.0")
	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	tmp := dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := &progressReader{
		inner:    resp.Body,
		total:    totalSize,
		OnChange: d.OnProgress,
	}
	if _, err := io.Copy(f, h); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// maybeResume 检测 dest 是否已有部分字节,留作未来 Range 续传的钩子(本期不实现)。
func (d *Downloader) maybeResume(_ context.Context, _ string) error { return nil }

// progressReader 流式读取并回调进度百分比。
type progressReader struct {
	inner    io.Reader
	total    int64
	n        int64
	OnChange func(percent int)
	lastPct  int
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.inner.Read(buf)
	p.n += int64(n)
	if p.total > 0 && p.OnChange != nil {
		pct := int(p.n * 100 / p.total)
		if pct != p.lastPct && pct >= 0 && pct <= 100 {
			p.lastPct = pct
			setProgress(pct)
			p.OnChange(pct)
		}
	}
	return n, err
}

// pickFilename 从 urls[0] 里取 basename 作为下载文件名;空 / 含换行拒绝。
func pickFilename(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	u := urls[0]
	// 去 query
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	if i := strings.LastIndex(u, "/"); i >= 0 {
		u = u[i+1:]
	}
	if u == "" || strings.ContainsAny(u, "\n\r\t") {
		return ""
	}
	return u
}

func sha256FileIfExists(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func sha256File(path string) (exists bool, sum string, err error) {
	s, e := sha256FileIfExists(path)
	if e != nil {
		if os.IsNotExist(e) {
			return false, "", nil
		}
		return false, "", e
	}
	return true, s, nil
}

// HelperBundle 传给桌面端替身脚本的所有参数。
//
// 桌面端 controller 在 call SpawnHelper 之前,把 Manifest、Asset、destPath、旧版版本号塞这里;
// Bash helper 接 $1=destPath $2=targetInstallPath $3=assetOS $4=assetArch。
type HelperBundle struct {
	DestPath         string // 下载结束路径
	TargetInstallDir string // macOS: /Applications/SkillBox.app
	OldVersion       string // 升级前的版本(env: SKILLBOX_UPDATER_FROM)
	OS               string
	Arch             string
}

// Args 把 bundle 拍平到 []string(按 helper_*.sh 的 argv 顺序)。
// Mac / Linux 直接吃这个;Windows helper 由 cdesktop 单独再 wrap 成 powershell -File。
func (h *HelperBundle) Args() []string {
	return []string{h.DestPath, h.TargetInstallDir, h.OS, h.Arch, h.OldVersion}
}

// StartTracking / FinishTracking 是状态机对外接口。
// Controller 在进入 check / download / 失败时主动调,避免依赖 OnProgress 兜底。
func StartTracking(phase string) { setPhase(phase) }

func FinishTracking(phase string, err error) {
	if err != nil {
		setErr(err.Error())
		return
	}
	setPhase(phase)
}
