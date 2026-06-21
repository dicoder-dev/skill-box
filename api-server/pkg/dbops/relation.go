package dbops

import (
	"fmt"
	"ginp-api/pkg/where"
)

// BuildJoinList 根据关联搜索配置和主实体构造JOIN列表
// mainEntity: 主实体实例，用于获取表名和主键
// relations: 关联搜索配置列表
// 返回: JOIN项目列表
type BuildJoinListConfig struct {
	MainEntity interface{}	//主实体实例，用于获取表名和主键
	Relations []*where.RelationSearch
	MainTableName string
	MainPrimaryKey string
}

func BuildJoinList(buildJoinListConfig *BuildJoinListConfig) []JoinItem {
	var joinList []JoinItem
	
	if buildJoinListConfig.Relations == nil || len(buildJoinListConfig.Relations) == 0 {
		return joinList
	}
	

	for _, relation := range buildJoinListConfig.Relations {
		// 构造JOIN条件，使用动态获取的表名和主键
		onCondition := fmt.Sprintf("%s.%s = %s.%s", buildJoinListConfig.MainTableName, buildJoinListConfig.MainPrimaryKey, relation.RelationName, relation.RelationFidName)
		
		joinList = append(joinList, JoinItem{
			Table:  relation.RelationName,
			On:     onCondition,
			Wheres: relation.Where,
		})
	}
	
	return joinList
}