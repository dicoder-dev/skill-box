package skillversion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"ginp-api/internal/skillversion/gitconfig"
)

// PushQueueItem 重试队列单条记录。
//
// 2026-07-17:push 失败时入队;启动时 + 任意成功 push 后扫一次重试。
// 上限 5 次,失败后从队列移除(用户手动重试入口是 SkillsView 的 "Push" 按钮)。
type PushQueueItem struct {
	Hash         string    `json:"hash"`
	Message      string    `json:"message"`
	LastError    string    `json:"last_error"`
	Attempts     int       `json:"attempts"`
	FirstAttempt time.Time `json:"first_attempt"`
	LastAttempt  time.Time `json:"last_attempt"`
}

const (
	maxPushAttempts  = 5
	pushQueueFile    = "git_push_queue.json"
	pushQueueVersion = 1
)

// pushQueue 全局单例,进程退出前 flush 到磁盘。
var pushQueue = newPushQueue()

type pushQueueImpl struct {
	mu    sync.Mutex
	items []PushQueueItem
	path  string
	dirty bool
}

func newPushQueue() *pushQueueImpl {
	q := &pushQueueImpl{
		items: []PushQueueItem{},
		path:  filepath.Join(gitconfig.DataDir(), pushQueueFile),
	}
	q.load()
	return q
}

// Add 入队一条 push 失败记录。
func (q *pushQueueImpl) Add(hash, msg, errStr string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	// 同一 hash 已存在 → 累加 attempts
	for i := range q.items {
		if q.items[i].Hash == hash {
			q.items[i].Attempts++
			q.items[i].LastError = errStr
			q.items[i].LastAttempt = now
			if q.items[i].Attempts >= maxPushAttempts {
				// 超过上限 → 移除(等用户手动重试)
				q.items = append(q.items[:i], q.items[i+1:]...)
			}
			q.dirty = true
			q.flushLocked()
			return
		}
	}
	q.items = append(q.items, PushQueueItem{
		Hash:         hash,
		Message:      msg,
		LastError:    errStr,
		Attempts:     1,
		FirstAttempt: now,
		LastAttempt:  now,
	})
	q.dirty = true
	q.flushLocked()
}

// Remove 成功 push 后移除对应 hash。
func (q *pushQueueImpl) Remove(hash string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.items {
		if q.items[i].Hash == hash {
			q.items = append(q.items[:i], q.items[i+1:]...)
			q.dirty = true
			q.flushLocked()
			return
		}
	}
}

// Len 队列长度(Status 接口用)。
func (q *pushQueueImpl) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Items 复制一份给前端展示(脱敏 hash 后 7 位)。
func (q *pushQueueImpl) Items() []PushQueueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]PushQueueItem, len(q.items))
	copy(out, q.items)
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastAttempt.After(out[j].LastAttempt)
	})
	return out
}

// Flush 进程退出前刷盘(bootstrap.RegisterExitHook 调用)。
func (q *pushQueueImpl) Flush() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.flushLocked()
}

func (q *pushQueueImpl) flushLocked() {
	if !q.dirty {
		return
	}
	type onDisk struct {
		Version int             `json:"version"`
		Items   []PushQueueItem `json:"items"`
	}
	payload := onDisk{Version: pushQueueVersion, Items: q.items}
	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(q.path), ".git_push_queue.tmp.*")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return
	}
	tmp.Close()
	os.Chmod(tmpPath, 0o600)
	os.Rename(tmpPath, q.path)
	q.dirty = false
}

func (q *pushQueueImpl) load() {
	b, err := os.ReadFile(q.path)
	if err != nil {
		return
	}
	var payload struct {
		Version int             `json:"version"`
		Items   []PushQueueItem `json:"items"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return
	}
	if payload.Version != pushQueueVersion {
		// 版本不兼容,清空
		return
	}
	q.items = payload.Items
	if q.items == nil {
		q.items = []PushQueueItem{}
	}
}

// RetryFailed 启动时扫一次重试队列(bootstrap.RegisterBootHook 调用)。
func RetryFailed(repo *Repo) {
	items := pushQueue.Items()
	for _, it := range items {
		if it.Attempts >= maxPushAttempts {
			continue
		}
		if err := repo.Push(); err == nil {
			pushQueue.Remove(it.Hash)
		}
	}
}