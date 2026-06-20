package ginp

// OperationType 接口操作类型
type OperationType string

const (
	// CRUD 基础操作
	OpCreate   OperationType = "CREATE"   // 新增/创建
	OpRead     OperationType = "READ"     // 查询/读取
	OpUpdate   OperationType = "UPDATE"   // 修改/更新
	OpDelete   OperationType = "DELETE"   // 删除
	OpSearch   OperationType = "SEARCH"   // 搜索/列表查询
	
	// 其他常见操作
	OpImport   OperationType = "IMPORT"   // 导入
	OpExport   OperationType = "EXPORT"   // 导出
	OpDownload OperationType = "DOWNLOAD" // 下载
	OpUpload   OperationType = "UPLOAD"   // 上传
	OpSync     OperationType = "SYNC"     // 同步
	OpAudit    OperationType = "AUDIT"    // 审核
	OpApprove  OperationType = "APPROVE"  // 批准
	OpReject   OperationType = "REJECT"   // 拒绝
	OpCancel   OperationType = "CANCEL"   // 取消

	//其他
	OpOther OperationType = "OTHER"
	//用户与自定义
	OpUserCustom OperationType = "USER_CUSTOM"
	

)

// String 返回操作类型的字符串表示
func (o OperationType) String() string {
	return string(o)
}
