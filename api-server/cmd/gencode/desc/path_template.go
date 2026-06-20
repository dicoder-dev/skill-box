package desc

import "path/filepath"

// 模板路径 entity
func TemplatePathEntity() string {
	return filepath.Join(GetDirTemplate(), "entity.tmpl")
}

// 模板路径 router
func TemplatePathRouter() string {
	return filepath.Join(GetDirTemplate(), "router.tmpl")
}

// 模板路径 controller
func TemplatePathController() string {
	return filepath.Join(GetDirTemplate(), "controller.tmpl")
}

// 模板路径 service
func TemplatePathService() string {
	return filepath.Join(GetDirTemplate(), "service.tmpl")
}

// 模板路径 model
func TemplatePathModel() string {
	return filepath.Join(GetDirTemplate(), "model.tmpl")
}

// 模板路径 fields
func TemplatePathFields() string {
	return filepath.Join(GetDirTemplate(), "fields.tmpl")
}

// 模板路径 add api
func TemplatePathAddApi() string {
	return filepath.Join(GetDirTemplate(), "addapi.tmpl")
}
