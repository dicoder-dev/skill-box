package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/project"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestService 构造一个用 sqlite 内存 DB 的 project.Service。
// 每次调用都是新库,适合单元测试隔离。
func newTestService(t *testing.T) *project.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.Project{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return project.New(db, db)
}

func TestCreate_Ok(t *testing.T) {
	svc := newTestService(t)
	p, err := svc.Create(&entity.Project{
		Name: "skill-box", Alias: "sb", RootPath: "/tmp/sb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero id")
	}
}

func TestCreate_TrimSpaces(t *testing.T) {
	svc := newTestService(t)
	p, err := svc.Create(&entity.Project{
		Name: "  trim me  ", Alias: "  trim  ", RootPath: "  /tmp/trim  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "trim me" || p.Alias != "trim" || p.RootPath != "/tmp/trim" {
		t.Errorf("trim failed: %+v", p)
	}
}

func TestCreate_EmptyFields(t *testing.T) {
	svc := newTestService(t)
	cases := []struct {
		name string
		in   *entity.Project
		want error
	}{
		{"empty name", &entity.Project{Alias: "a", RootPath: "/x"}, project.ErrEmptyName},
		{"empty alias", &entity.Project{Name: "n", RootPath: "/x"}, project.ErrEmptyAlias},
		{"empty root", &entity.Project{Name: "n", Alias: "a"}, project.ErrEmptyRoot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Create(c.in)
			if !errors.Is(err, c.want) {
				t.Errorf("got %v, want %v", err, c.want)
			}
		})
	}
}

func TestCreate_AliasUnique(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(&entity.Project{Name: "a", Alias: "x", RootPath: "/p1"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(&entity.Project{Name: "b", Alias: "x", RootPath: "/p2"})
	if !errors.Is(err, project.ErrAliasExists) {
		t.Errorf("got %v, want ErrAliasExists", err)
	}
}

func TestCreate_RootUnique(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(&entity.Project{Name: "a", Alias: "x", RootPath: "/p"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(&entity.Project{Name: "b", Alias: "y", RootPath: "/p"})
	if !errors.Is(err, project.ErrRootExists) {
		t.Errorf("got %v, want ErrRootExists", err)
	}
}

func TestUpdate_Ok(t *testing.T) {
	svc := newTestService(t)
	p, _ := svc.Create(&entity.Project{Name: "n", Alias: "a", RootPath: "/r"})
	upd, err := svc.Update(p.ID, &entity.Project{Description: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Description != "hello" {
		t.Errorf("desc: %+v", upd)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Update(999, &entity.Project{Name: "n"})
	if !errors.Is(err, project.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestUpdate_AliasConflict(t *testing.T) {
	svc := newTestService(t)
	p1, _ := svc.Create(&entity.Project{Name: "n1", Alias: "a", RootPath: "/r1"})
	svc.Create(&entity.Project{Name: "n2", Alias: "b", RootPath: "/r2"})
	_, err := svc.Update(p1.ID, &entity.Project{Alias: "b"})
	if !errors.Is(err, project.ErrAliasExists) {
		t.Errorf("got %v, want ErrAliasExists", err)
	}
}

func TestGetByID(t *testing.T) {
	svc := newTestService(t)
	p, _ := svc.Create(&entity.Project{Name: "n", Alias: "a", RootPath: "/r"})
	got, err := svc.GetByID(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "n" {
		t.Errorf("name=%q", got.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.GetByID(999)
	if !errors.Is(err, project.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestList_PaginationAndKeyword(t *testing.T) {
	svc := newTestService(t)
	// 真实落盘到临时目录以避开路径唯一约束
	tmp := t.TempDir()
	for i := 0; i < 5; i++ {
		_, err := svc.Create(&entity.Project{
			Name:     "proj-" + string(rune('a'+i)),
			Alias:    "a" + string(rune('a'+i)),
			RootPath: filepath.Join(tmp, string(rune('a'+i))),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// 关键字
	got, err := svc.List(project.ListQuery{Keyword: "proj-c"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 {
		t.Errorf("keyword total=%d", got.Total)
	}
	// 分页
	got, err = svc.List(project.ListQuery{Page: 1, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 5 || len(got.Items) != 2 {
		t.Errorf("paged total=%d items=%d", got.Total, len(got.Items))
	}
}

func TestDelete(t *testing.T) {
	svc := newTestService(t)
	p, _ := svc.Create(&entity.Project{Name: "n", Alias: "a", RootPath: "/r"})
	if err := svc.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetByID(p.ID); !errors.Is(err, project.ErrNotFound) {
		t.Errorf("after delete: got %v, want ErrNotFound", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Delete(999); !errors.Is(err, project.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

// TestService_PhysicalRootNotRequired 验证物理根不存在时仍可创建(占位语义)。
func TestService_PhysicalRootNotRequired(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Create(&entity.Project{
		Name: "future", Alias: "f", RootPath: "/this/path/does/not/exist/yet",
	})
	if err != nil {
		t.Errorf("physical root should not be required at create time, got %v", err)
	}
	_ = os.Getenv // 防止 import 被优化
}
