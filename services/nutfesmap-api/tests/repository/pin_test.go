package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	repository "nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func newMockPin(t *testing.T) (*repository.PinRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	r := repository.NewPinRepository(db)
	cleanup := func() { _ = db.Close() }
	return r, mock, cleanup
}

// ------------------------------------------------------------
// SQL（リポジトリ実装と完全一致）
// ------------------------------------------------------------

const selectPinsByMapSQL = `
		SELECT
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm, place,
		  category, status, wait_minutes, created_at, modified_at
	FROM pins
		WHERE map_id = ? 
		ORDER BY modified_at DESC, id ASC
	`

const selectPinByIDSQL = `
		SELECT
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm, place,
		  category, status, wait_minutes, created_at, modified_at
	FROM pins
		WHERE id = ?
		LIMIT 1
	`

const insertPinSQL = `
		INSERT INTO pins (
		  id, map_id, name, description, description_image,
		  type, link_to_map_id, x_norm, y_norm, place,
		  category, status, wait_minutes, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`

// ------------------------------------------------------------
// FindByMapID
// ------------------------------------------------------------

func TestPinRepository_FindByMapID_OK(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	rows := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "屋台A", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.25, 0.5, sql.NullString{Valid: false}, "food", "open", 0, now, now).
		AddRow("p2", "m1", "案内", sql.NullString{Valid: true, String: "受付はこちら"}, sql.NullString{Valid: false},
			"info", sql.NullString{Valid: false}, 0.8, 0.1, sql.NullString{Valid: true, String: "エリアA"}, "service", "paused", 5, now, now)

	mock.ExpectQuery(selectPinsByMapSQL).
		WithArgs("m1").
		WillReturnRows(rows)

	list, err := r.FindByMapID(context.Background(), "m1")
	if err != nil {
		t.Fatalf("FindByMapID error: %v", err)
	}
	if len(list) != 2 || list[0].ID != "p1" || list[1].Name != "案内" {
		t.Fatalf("unexpected list: %#v", list)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// FindByID
// ------------------------------------------------------------

func TestPinRepository_FindByID_OK(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	row := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "屋台A",
			sql.NullString{Valid: true, String: "焼きそば"},
			sql.NullString{Valid: true, String: "BASE64PNG"},
			"exhibit", sql.NullString{Valid: true, String: "m_link"}, 0.2, 0.3,
			sql.NullString{Valid: true, String: "講義棟前"},
			"food", "open", 0, now, now)

	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(row)

	got, err := r.FindByID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if got == nil || got.ID != "p1" || got.MapID != "m1" || got.Type != "exhibit" {
		t.Fatalf("unexpected pin: %#v", got)
	}
	if got.Description == nil || *got.Description != "焼きそば" || got.DescriptionImage == nil {
		t.Fatalf("unexpected nullable fields: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_FindByID_NotFound(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(cols))

	got, err := r.FindByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("FindByID unexpected err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// CreateOnMap
// ------------------------------------------------------------

func TestPinRepository_CreateOnMap_OK(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	req := &repository.PinCreateRequest{
		Name:        "ステージ",
		XNorm:       0.5,
		YNorm:       0.6,
		Place:       strPtr("中央広場"),
		Category:    "plan",
		Type:        strPtr("info"),
		Status:      strPtr("open"),
		WaitMinutes: intPtr(0),
	}

	// INSERT
	mock.ExpectExec(insertPinSQL).
		WithArgs("new_pin", "map_1", req.Name, (*string)(nil), (*string)(nil),
			"info", (*string)(nil), 0.5, 0.6, "中央広場", "plan", "open", 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 直後の再取得
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	row := sqlmock.NewRows(cols).
		AddRow("new_pin", "map_1", "ステージ",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"info", sql.NullString{Valid: false}, 0.5, 0.6,
			sql.NullString{Valid: true, String: "中央広場"},
			"plan", "open", 0, now, now)
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("new_pin").
		WillReturnRows(row)

	got, err := r.CreateOnMap(context.Background(), "new_pin", "map_1", req)
	if err != nil {
		t.Fatalf("CreateOnMap error: %v", err)
	}
	if got == nil || got.ID != "new_pin" || got.Category != "plan" || got.Type != "info" {
		t.Fatalf("unexpected created pin: %#v", got)
	}
	if got.Place == nil || *got.Place != "中央広場" {
		t.Fatalf("expected place to be set, got %#v", got.Place)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_CreateOnMap_ValidationError(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	// xNorm out of range -> エラー（SQL発行なし）
	req := &repository.PinCreateRequest{
		Name:     "NG",
		XNorm:    1.2,
		YNorm:    0.3,
		Category: "food",
	}
	if _, err := r.CreateOnMap(context.Background(), "id", "map", req); err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil { // 期待しているクエリは無し
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// UpdatePartial
// ------------------------------------------------------------

func TestPinRepository_UpdatePartial_UpdateSome_OK(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	// 1) 現在値取得
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "旧名",
			sql.NullString{Valid: true, String: "説明あり"},
			sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.1, 0.2,
			sql.NullString{Valid: true, String: "東門付近"},
			"food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(curRow)

	// 2) 更新（name, description=NULL, x_norm, status, wait_minutes）
	updateSQL := "UPDATE pins SET name = ?, description = NULL, x_norm = ?, status = ?, wait_minutes = ?, modified_at = ? WHERE id = ?"
	mock.ExpectExec(updateSQL).
		WithArgs("新名", 0.3, "paused", 10, sqlmock.AnyArg(), "p1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3) 再取得
	afterRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "新名",
			sql.NullString{Valid: false},
			sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.3, 0.2,
			sql.NullString{Valid: true, String: "東門付近"},
			"food", "paused", 10, now, now.Add(time.Minute))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(afterRow)

	req := &repository.PinUpdateRequest{
		Name:        repository.OptionalString{Set: true, Value: "新名"},
		Description: repository.OptionalNullableString{Set: true, Null: true},
		XNorm:       repository.OptionalFloat64{Set: true, Value: 0.3},
		Status:      repository.OptionalString{Set: true, Value: "paused"},
		WaitMinutes: repository.OptionalInt{Set: true, Value: 10},
	}

	got, err := r.UpdatePartial(context.Background(), "p1", req)
	if err != nil {
		t.Fatalf("UpdatePartial error: %v", err)
	}
	if got == nil || got.Name != "新名" || got.Status != "paused" || got.WaitMinutes != 10 || got.XNorm != 0.3 {
		t.Fatalf("unexpected updated pin: %#v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_UpdatePartial_InvalidXNorm_Error(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	// 現在値取得（UpdatePartial は最初に FindByID を呼ぶ）
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "旧名",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.1, 0.2,
			sql.NullString{Valid: false},
			"food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(curRow)

	req := &repository.PinUpdateRequest{
		XNorm: repository.OptionalFloat64{Set: true, Value: 1.5}, // invalid
	}

	if _, err := r.UpdatePartial(context.Background(), "p1", req); err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	// 再取得やUPDATEは呼ばれない
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_UpdatePartial_NoChange_ReturnCur(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "Name",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.1, 0.2,
			sql.NullString{Valid: false},
			"food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(curRow)

	req := &repository.PinUpdateRequest{} // 変更なし

	got, err := r.UpdatePartial(context.Background(), "p1", req)
	if err != nil {
		t.Fatalf("UpdatePartial error: %v", err)
	}
	if got == nil || got.ID != "p1" || got.Name != "Name" {
		t.Fatalf("unexpected result for no-change: %#v", got)
	}

	// UPDATE/再取得は行われない
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_UpdatePartial_UpdatePlace(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p2", "m1", "Name",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.4, 0.5,
			sql.NullString{Valid: false},
			"food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p2").
		WillReturnRows(curRow)

	updateSQL := "UPDATE pins SET place = ?, modified_at = ? WHERE id = ?"
	mock.ExpectExec(updateSQL).
		WithArgs("新会場", sqlmock.AnyArg(), "p2").
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(cols).
		AddRow("p2", "m1", "Name",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.4, 0.5,
			sql.NullString{Valid: true, String: "新会場"},
			"food", "open", 0, now, now.Add(time.Minute))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p2").
		WillReturnRows(afterRow)

	req := &repository.PinUpdateRequest{
		Place: repository.OptionalNullableString{Set: true, Value: "新会場"},
	}

	got, err := r.UpdatePartial(context.Background(), "p2", req)
	if err != nil {
		t.Fatalf("UpdatePartial error: %v", err)
	}
	if got == nil || got.Place == nil || *got.Place != "新会場" {
		t.Fatalf("expected place to update, got %#v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_UpdatePartial_ClearPlace(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p3", "m1", "Name",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.2, 0.3,
			sql.NullString{Valid: true, String: "旧会場"},
			"food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p3").
		WillReturnRows(curRow)

	updateSQL := "UPDATE pins SET place = NULL, modified_at = ? WHERE id = ?"
	mock.ExpectExec(updateSQL).
		WithArgs(sqlmock.AnyArg(), "p3").
		WillReturnResult(sqlmock.NewResult(0, 1))

	afterRow := sqlmock.NewRows(cols).
		AddRow("p3", "m1", "Name",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.2, 0.3,
			sql.NullString{Valid: false},
			"food", "open", 0, now, now.Add(time.Minute))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p3").
		WillReturnRows(afterRow)

	req := &repository.PinUpdateRequest{
		Place: repository.OptionalNullableString{Set: true, Null: true},
	}

	got, err := r.UpdatePartial(context.Background(), "p3", req)
	if err != nil {
		t.Fatalf("UpdatePartial error: %v", err)
	}
	if got == nil || got.Place != nil {
		t.Fatalf("expected place to be cleared, got %#v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// Delete
// ------------------------------------------------------------

func TestPinRepository_Delete_OK(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM pins WHERE id = ?").
		WithArgs("p1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := r.Delete(context.Background(), "p1"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinRepository_Delete_NotFound(t *testing.T) {
	r, mock, cleanup := newMockPin(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM pins WHERE id = ?").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := r.Delete(context.Background(), "missing"); err == nil {
		t.Fatalf("expected sql.ErrNoRows, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// helpers (local)
// ------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }
