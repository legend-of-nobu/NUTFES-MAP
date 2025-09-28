package repository_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"nutfesmap/internal/model"
	repo "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMock(t *testing.T) (*repo.MapRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	repo := repo.NewMapRepository(db)
	cleanup := func() {
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func TestMapRepository_Insert_OK(t *testing.T) {
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	// Arrange
	m := &model.Map{
		ID:            "map_123",
		Name:          "Campus 2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		ParentMapID:   nil,
		HasFloors:     true,
		FloorCount:    3,
	}

	// Execの引数のうち created_at / modified_at は time.Now().UTC() なので AnyArg() で受ける
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			m.ID, m.Name, m.ImageData, m.NaturalWidth, m.NaturalHeight,
			m.ParentMapID, m.HasFloors, m.FloorCount, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Act
	err := repo.Insert(context.Background(), m)

	// Assert
	if err != nil {
		t.Fatalf("Insert returned err: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("there were unfulfilled expectations: %v", err)
	}
}

func TestMapRepository_Insert_ExecError(t *testing.T) {
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	m := &model.Map{
		ID:            "map_123",
		Name:          "Campus 2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		HasFloors:     false,
		FloorCount:    0,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			m.ID, m.Name, m.ImageData, m.NaturalWidth, m.NaturalHeight,
			m.ParentMapID, m.HasFloors, m.FloorCount, sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(assertErr("insert failed"))

	err := repo.Insert(context.Background(), m)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindAggregate_OK(t *testing.T) {
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	// 1) 本体行
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	now := time.Now().UTC()
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		id, "Campus 2025", "iVBORw0K...", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(mainRow)

	// 2) 子件数
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 3) 子一覧
	childCols := []string{"id", "name", "has_floors", "floor_count"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("map_201", "1F", false, 0).
		AddRow("map_202", "2F", false, 0)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs(id).
		WillReturnRows(childRows)

	// Act
	agg, err := repo.FindAggregate(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("FindAggregate returned err: %v", err)
	}
	if agg == nil || agg.Base == nil {
		t.Fatalf("FindAggregate returned nil aggregate")
	}
	if agg.Base.ID != id || agg.ChildrenCount != 2 || len(agg.Children) != 2 {
		t.Fatalf("unexpected aggregate: %#v", agg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindAggregate_NoRows(t *testing.T) {
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_not_found"
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	// 0行を返す → sql.ErrNoRows になる
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(mainCols))

	agg, err := repo.FindAggregate(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if agg != nil {
		t.Fatalf("expected nil, got: %#v", agg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMapRepository_FindAggregate_QueryError(t *testing.T) {
	repo, mock, cleanup := newMock(t)
	defer cleanup()

	id := "map_123"
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(id).
		WillReturnError(assertErr("select failed"))

	agg, err := repo.FindAggregate(context.Background(), id)
	if err == nil {
		t.Fatalf("expected error, got nil; agg=%#v", agg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// --- helpers ---

// assertErr はテスト内で分かりやすい固定エラーを作る小道具
type assertErr string

func (e assertErr) Error() string { return string(e) }
