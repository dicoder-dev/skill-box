package ginp

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

const maxLogBodyLen = 3072

// track first request per path per day to avoid truncation once per day.
var (
	firstSeenMu    sync.Mutex
	firstSeenDay   string
	firstSeenPaths = map[string]struct{}{}
)

// LoggingMiddleware 请求日志中间件
// 记录每个请求的方法、路径、状态码和耗时
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()
		requestTime := startTime

		// capture request body without consuming it
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// wrap response writer to capture response body
		bodyWriter := &responseBodyWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = bodyWriter

		// 处理请求
		c.Next()

		// 记录响应信息
		statusCode := c.Writer.Status()
		duration := time.Since(startTime)
		respCode, respMsg := extractCodeAndMsgFromBody(bodyWriter.body.Bytes())

		// try to fetch user id if available
		cp := &ContextPlus{Context: c}
		userID := cp.GetUserID()

		// capture token from Authorization header (if present)
		authHeader := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		if showLog {
			fmt.Printf("[%s] %s %s |  %d |  %v | Msg: %v | Code: %v\n",
				clientIP,
				method,
				path,
				statusCode,
				duration,
				respMsg,
				respCode,
			)
		}

		// append detailed request/response info into log file
		go func(entry requestLogEntry) {
			_ = writeRequestLog(entry)
			_ = writeDailyAPIStats(entry)
		}(requestLogEntry{
			RequestTime:  requestTime,
			ClientIP:     clientIP,
			Method:       method,
			Path:         path,
			RequestQuery: c.Request.URL.RawQuery,
			Status:       statusCode,
			Duration:     duration,
			UserID:       userID,
			Token:        token,
			UserAgent:    c.Request.UserAgent(),
			Referer:      c.Request.Referer(),
			HasAuth:      authHeader != "",
			RequestBody:  prepareBodyForLog(path, string(requestBody)),
			ResponseBody: prepareBodyForLog(path, responseBodyForLog(statusCode, bodyWriter.body.String())),
			RespCode:     respCode,
			RespMsg:      respMsg,
		})
	}
}

// responseBodyWriter captures response body content while delegating to original writer.
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// requestLogEntry holds detailed request information for persistence.
type requestLogEntry struct {
	RequestTime  time.Time
	ClientIP     string
	Method       string
	Path         string
	Status       int
	Duration     time.Duration
	RequestQuery string
	RequestBody  string
	ResponseBody string
	UserID       uint
	Token        string
	UserAgent    string
	Referer      string
	HasAuth      bool
	RespCode     any
	RespMsg      string
}

var requestLogMu sync.Mutex
var dailyAPIStatsMu sync.Mutex

// requestLogBaseDir 返回请求日志/统计文件的根目录。
// 优先用 logger.SetLogPath 设过的绝对路径(桌面端 ~/.<AppName>/logs),否则兜底 ./logs。
func requestLogBaseDir() string {
	base := strings.TrimRight(logger.GetLogPath(), "/")
	if base == "" {
		base = "logs"
	}
	return base
}

// getRequestLogFilename returns one of the three daily request log files:
// 1. /api/parameter and /api/like -> MM-DDcanshu_request.txt
// 2. other 200 requests -> MM-DD-request.txt
// 3. non-200 requests -> MM-DD-error_request.txt
func getRequestLogFilename(path string, status int, now time.Time) string {
	date := now.Format("01-02")

	if status != 200 {
		return fmt.Sprintf("%s-error_request.txt", date)
	}

	if strings.HasPrefix(path, "/api/parameter") || strings.HasPrefix(path, "/api/like") {
		return fmt.Sprintf("%s-canshu_request.txt", date)
	}

	return fmt.Sprintf("%s-request.txt", date)
}

// writeRequestLog persists request details to ./logs/<month>/<daily log file>.
func writeRequestLog(entry requestLogEntry) error {
	requestLogMu.Lock()
	defer requestLogMu.Unlock()

	// ensure strings are valid UTF-8 so editors can open the file
	entry.RequestBody = sanitizeForLog(entry.RequestBody)
	entry.RequestQuery = sanitizeForLog(entry.RequestQuery)
	entry.ResponseBody = sanitizeForLog(entry.ResponseBody)
	entry.UserAgent = sanitizeForLog(entry.UserAgent)
	entry.Referer = sanitizeForLog(entry.Referer)
	entry.RespMsg = sanitizeForLog(entry.RespMsg)

	// 按月份创建目录：logs/YYYY-MM/
	logDir := filepath.Join(requestLogBaseDir(), time.Now().Format("2006-01"))
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	// 每天最多 3 个文件：参数/点赞、普通请求、错误请求
	filename := getRequestLogFilename(entry.Path, entry.Status, time.Now())
	filePath := filepath.Join(logDir, filename)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s][%s] %s %d\n",
		entry.RequestTime.Format("2006-01-02 15:04:05"),
		entry.Method,
		entry.Path,
		entry.Status,
	)
	fmt.Fprintf(&sb, "IP:%s  user_id:%v  耗时:%s\n",
		entry.ClientIP,
		entry.UserID,
		entry.Duration,
	)
	fmt.Fprintf(&sb, "User-Agent:%s  Referer:%s  Authorization:%t\n",
		emptyLogValue(entry.UserAgent),
		emptyLogValue(entry.Referer),
		entry.HasAuth,
	)
	if entry.RequestQuery != "" {
		fmt.Fprintf(&sb, "请求参数:\n %s\n", entry.RequestQuery)
	}
	if entry.RequestBody != "" {
		fmt.Fprintf(&sb, "请求体:\n %s\n", entry.RequestBody)
	}
	if entry.ResponseBody != "" {
		fmt.Fprintf(&sb, "响应体:\n %s\n", entry.ResponseBody)
	}
	sb.WriteString("\n")

	_, err = f.WriteString(sb.String())
	return err
}

// writeDailyAPIStats persists sorted daily counters under ./logs/<month>/.
func writeDailyAPIStats(entry requestLogEntry) error {
	if entry.Method == "OPTIONS" {
		return nil
	}

	dailyAPIStatsMu.Lock()
	defer dailyAPIStatsMu.Unlock()

	logDir := filepath.Join(requestLogBaseDir(), entry.RequestTime.Format("2006-01"))
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	date := entry.RequestTime.Format("01-02")
	apiStatsPath := filepath.Join(logDir, fmt.Sprintf("stats_api_%s.csv", date))
	if err := updateAPIStatsFile(apiStatsPath, entry.Method, entry.Path); err != nil {
		return err
	}

	if entry.UserID > 0 {
		userStatsPath := filepath.Join(logDir, fmt.Sprintf("stats_user_api_%s.csv", date))
		if err := updateUserAPIStatsFile(userStatsPath, entry.UserID); err != nil {
			return err
		}
	}

	return nil
}

type apiStatsRow struct {
	Method string
	Path   string
	Count  int
}

func updateAPIStatsFile(filePath, method, path string) error {
	rows, err := readAPIStatsRows(filePath)
	if err != nil {
		return err
	}

	key := method + "\x00" + path
	if row, ok := rows[key]; ok {
		row.Count++
	} else {
		rows[key] = &apiStatsRow{
			Method: method,
			Path:   path,
			Count:  1,
		}
	}

	ordered := make([]*apiStatsRow, 0, len(rows))
	for _, row := range rows {
		ordered = append(ordered, row)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Count != ordered[j].Count {
			return ordered[i].Count > ordered[j].Count
		}
		if ordered[i].Path != ordered[j].Path {
			return ordered[i].Path < ordered[j].Path
		}
		return ordered[i].Method < ordered[j].Method
	})

	records := [][]string{{"排名", "请求方法", "接口路径", "请求次数"}}
	for i, row := range ordered {
		records = append(records, []string{
			strconv.Itoa(i + 1),
			row.Method,
			row.Path,
			strconv.Itoa(row.Count),
		})
	}
	return writeCSVFile(filePath, records)
}

func readAPIStatsRows(filePath string) (map[string]*apiStatsRow, error) {
	rows := map[string]*apiStatsRow{}
	records, err := readCSVFile(filePath)
	if err != nil {
		return nil, err
	}
	for i, record := range records {
		if i == 0 || len(record) < 4 {
			continue
		}
		count, err := strconv.Atoi(record[3])
		if err != nil {
			continue
		}
		method := record[1]
		path := record[2]
		rows[method+"\x00"+path] = &apiStatsRow{
			Method: method,
			Path:   path,
			Count:  count,
		}
	}
	return rows, nil
}

type userAPIStatsRow struct {
	UserID uint
	Count  int
}

func updateUserAPIStatsFile(filePath string, userID uint) error {
	rows, err := readUserAPIStatsRows(filePath)
	if err != nil {
		return err
	}

	if row, ok := rows[userID]; ok {
		row.Count++
	} else {
		rows[userID] = &userAPIStatsRow{
			UserID: userID,
			Count:  1,
		}
	}

	ordered := make([]*userAPIStatsRow, 0, len(rows))
	for _, row := range rows {
		ordered = append(ordered, row)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Count != ordered[j].Count {
			return ordered[i].Count > ordered[j].Count
		}
		return ordered[i].UserID < ordered[j].UserID
	})

	records := [][]string{{"排名", "用户ID", "请求次数"}}
	for i, row := range ordered {
		records = append(records, []string{
			strconv.Itoa(i + 1),
			strconv.FormatUint(uint64(row.UserID), 10),
			strconv.Itoa(row.Count),
		})
	}
	return writeCSVFile(filePath, records)
}

func readUserAPIStatsRows(filePath string) (map[uint]*userAPIStatsRow, error) {
	rows := map[uint]*userAPIStatsRow{}
	records, err := readCSVFile(filePath)
	if err != nil {
		return nil, err
	}
	for i, record := range records {
		if i == 0 || len(record) < 3 {
			continue
		}
		userID, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(record[2])
		if err != nil {
			continue
		}
		rows[uint(userID)] = &userAPIStatsRow{
			UserID: uint(userID),
			Count:  count,
		}
	}
	return rows, nil
}

func readCSVFile(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}

func writeCSVFile(filePath string, records [][]string) error {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	if err := writer.WriteAll(records); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

// extractCodeAndMsgFromBody tries to unmarshal response body JSON to fetch code/msg.
func extractCodeAndMsgFromBody(body []byte) (any, string) {
	if len(body) == 0 {
		return nil, ""
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err == nil {
		return data["code"], fmt.Sprint(data["msg"])
	}
	return nil, string(body)
}

func responseBodyForLog(statusCode int, body string) string {
	if body != "" {
		return body
	}
	if statusCode >= 400 {
		switch statusCode {
		case 404:
			return "Not Found"
		case 405:
			return "Method Not Allowed"
		default:
			return fmt.Sprintf("HTTP %d", statusCode)
		}
	}
	return ""
}

// prepareBodyForLog decides whether to truncate body: first request per path per day is kept full.
func prepareBodyForLog(path, body string) string {
	if body == "" {
		return body
	}

	today := time.Now().Format("2006-01-02")

	firstSeenMu.Lock()
	if firstSeenDay != today {
		firstSeenDay = today
		firstSeenPaths = map[string]struct{}{}
	}
	_, seen := firstSeenPaths[path]
	if !seen {
		firstSeenPaths[path] = struct{}{}
	}
	firstSeenMu.Unlock()

	if !seen {
		return body // first time today, do not truncate
	}

	if len(body) <= maxLogBodyLen {
		return body
	}
	return body[:maxLogBodyLen] + fmt.Sprintf("...(截断，原长度=%d,保留=%d)", len(body), maxLogBodyLen)
}

// sanitizeForLog replaces invalid UTF-8 runes to keep log files readable.
func sanitizeForLog(s string) string {
	if s == "" {
		return s
	}
	return string(bytes.ToValidUTF8([]byte(s), []byte{'.'}))
}

func emptyLogValue(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// CORSMiddleware CORS 跨域中间件
// 配置跨域允许的方法、头部和来源
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RecoveryMiddleware 恢复中间件
// 捕获 panic 异常，返回 500 错误而不是崩溃
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if showLog {
					log.Printf("[PANIC] %v", err)
				}
				c.JSON(500, gin.H{
					"code": codeFail,
					"msg":  "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

// RequestIDMiddleware 请求 ID 中间件
// 为每个请求生成唯一的 ID，便于追踪日志
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("X-Request-ID", requestID)
		}
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成唯一的请求 ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
