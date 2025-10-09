package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nutfesmap/internal/handlers"
	"nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
)

// ------------------------------------------------------------
// setup
// ------------------------------------------------------------

func setupEchoWithMockPin(t *testing.T) (*echo.Echo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	pinRepo := repository.NewPinRepository(db)
	pinH := handlers.NewPinHandler(pinRepo)

	e := echo.New()
	// ルーティング（本番と同じパス構成）
	e.GET("/maps/:mapId/pins", pinH.ListByMap)
	e.POST("/maps/:mapId/pins", pinH.CreateOnMap)
	e.GET("/pins/:pinId", pinH.Show)
	e.PATCH("/pins/:pinId", pinH.Update)
	e.DELETE("/pins/:pinId", pinH.Delete)

	cleanup := func() { _ = db.Close() }
	return e, mock, cleanup
}

// ------------------------------------------------------------
// SQL（PinRepository実装と完全一致）
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
// GET /maps/:mapId/pins
// ------------------------------------------------------------

func TestPinHandler_ListByMap_OK_SortedAndETag(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	// 並びは modified_at DESC / id ASC だが、ハンドラーで Name 昇順に安定化
	rows := sqlmock.NewRows(cols).
		AddRow("p2", "m1", "Zulu", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.6, 0.1, sql.NullString{Valid: true, String: "会場B"}, "other", "open", 0, now, now).
		AddRow("p1", "m1", "Alpha", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"info", sql.NullString{Valid: false}, 0.2, 0.3, sql.NullString{Valid: true, String: "会場A"}, "service", "paused", 3, now, now)

	mock.ExpectQuery(selectPinsByMapSQL).
		WithArgs("m1").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/maps/m1/pins", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	var got []*repository.PinResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json error: %v body=%s", err, rec.Body.String())
	}
	if len(got) != 2 || got[0].Name != "Alpha" || got[1].Name != "Zulu" {
		t.Fatalf("expected sorted by name asc, got: %#v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPinHandler_ListByMap_SQL_Error(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	mock.ExpectQuery(selectPinsByMapSQL).
		WithArgs("m1").
		WillReturnError(errors.New("select error"))

	req := httptest.NewRequest(http.MethodGet, "/maps/m1/pins", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// ハンドラーは err を返し、Echo が 500 にマップ
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// POST /maps/:mapId/pins
// ------------------------------------------------------------

func TestPinHandler_CreateOnMap_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	body := map[string]any{
		"name":     "受付",
		"xNorm":    0.4,
		"yNorm":    0.8,
		"place":    "学生ホール",
		"category": "service",
		"type":     "info",
		"status":   "open",
	}
	b, _ := json.Marshal(body)

	// INSERT（newID は uuid のため AnyArg でマッチ）
	mock.ExpectExec(insertPinSQL).
		WithArgs(sqlmock.AnyArg(), "m1", "受付", (*string)(nil), (*string)(nil),
			"info", (*string)(nil), 0.4, 0.8, "学生ホール", "service", "open", 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 再取得（ID は new_pin として返却してOK）
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	row := sqlmock.NewRows(cols).
		AddRow("new_pin", "m1", "受付",
			sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"info", sql.NullString{Valid: false}, 0.4, 0.8,
			sql.NullString{Valid: true, String: "学生ホール"},
			"service", "open", 0, now, now)

	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(row)

	req := httptest.NewRequest(http.MethodPost, "/maps/m1/pins", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("ETag must be set")
	}
	var created repository.PinResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("json error: %v", err)
	}
	if created.ID != "new_pin" || created.MapID != "m1" || created.Type != "info" || created.Category != "service" || created.Place == nil || *created.Place != "学生ホール" {
		t.Fatalf("unexpected created: %+v", created)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_CreateOnMap_BadJSON(t *testing.T) {
	e, _, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/maps/m1/pins", bytes.NewReader([]byte(`{"name":`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPinHandler_CreateOnMap_ValidationError_FromRepo(t *testing.T) {
	e, _, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	// xNorm が 1.2（リポジトリ側で早期バリデーションして SQL は発行されない）
	body := map[string]any{
		"name":     "NG",
		"xNorm":    1.2,
		"yNorm":    0.5,
		"category": "food",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/maps/m1/pins", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ------------------------------------------------------------
// GET /pins/:pinId
// ------------------------------------------------------------

func TestPinHandler_Show_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	row := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "受付", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"info", sql.NullString{Valid: false}, 0.4, 0.8, sql.NullString{Valid: true, String: "学生ホール"}, "service", "open", 0, now, now)

	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(row)

	req := httptest.NewRequest(http.MethodGet, "/pins/p1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("ETag must be set")
	}
	var got repository.PinResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.ID != "p1" || got.Name != "受付" || got.Place == nil || *got.Place != "学生ホール" {
		t.Fatalf("unexpected body: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_Show_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(cols))

	req := httptest.NewRequest(http.MethodGet, "/pins/missing", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_Show_SQL_Error(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnError(errors.New("select failed"))

	req := httptest.NewRequest(http.MethodGet, "/pins/p1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// PATCH /pins/:pinId
// ------------------------------------------------------------

func TestPinHandler_Update_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	// 1) 現在値
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "旧名", sql.NullString{Valid: true, String: "説明"},
			sql.NullString{Valid: false}, "exhibit", sql.NullString{Valid: false},
			0.1, 0.2, sql.NullString{Valid: true, String: "旧エリア"}, "food", "open", 0, now.Add(-time.Hour), now.Add(-time.Hour))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(curRow)

	// 2) UPDATE（name, description=NULL, x_norm, place, status, wait_minutes）
	updateSQL := "UPDATE pins SET name = ?, description = NULL, x_norm = ?, place = ?, status = ?, wait_minutes = ?, modified_at = ? WHERE id = ?"
	mock.ExpectExec(updateSQL).
		WithArgs("新名", 0.3, "新エリア", "paused", 10, sqlmock.AnyArg(), "p1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3) 再取得
	afterRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "新名", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.3, 0.2, sql.NullString{Valid: true, String: "新エリア"}, "food", "paused", 10, now, now.Add(time.Minute))
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(afterRow)

	body := map[string]any{
		"name":        "新名",
		"description": nil, // NULL 指定
		"xNorm":       0.3,
		"place":       "新エリア",
		"status":      "paused",
		"waitMinutes": 10,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/pins/p1", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("ETag must be set")
	}
	var got repository.PinResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Name != "新名" || got.Status != "paused" || got.WaitMinutes != 10 || got.XNorm != 0.3 || got.Place == nil || *got.Place != "新エリア" {
		t.Fatalf("unexpected updated: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_Update_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows(cols))

	req := httptest.NewRequest(http.MethodPatch, "/pins/missing", bytes.NewReader([]byte(`{"name":"X"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_Update_ValidationError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	// FindByID は呼ばれるので1回分返す
	now := time.Now().UTC()
	cols := []string{
		"id", "map_id", "name", "description", "description_image",
		"type", "link_to_map_id", "x_norm", "y_norm", "place",
		"category", "status", "wait_minutes", "created_at", "modified_at",
	}
	curRow := sqlmock.NewRows(cols).
		AddRow("p1", "m1", "旧名", sql.NullString{Valid: false}, sql.NullString{Valid: false},
			"exhibit", sql.NullString{Valid: false}, 0.1, 0.2, sql.NullString{Valid: false}, "food", "open", 0, now, now)
	mock.ExpectQuery(selectPinByIDSQL).
		WithArgs("p1").
		WillReturnRows(curRow)

	// xNorm が不正 → リポジトリでエラー、UPDATEは出ない
	req := httptest.NewRequest(http.MethodPatch, "/pins/p1", bytes.NewReader([]byte(`{"xNorm":1.5}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ------------------------------------------------------------
// DELETE /pins/:pinId
// ------------------------------------------------------------

func TestPinHandler_Delete_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM pins WHERE id = ?").
		WithArgs("p1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodDelete, "/pins/p1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPinHandler_Delete_NotFound(t *testing.T) {
	e, mock, cleanup := setupEchoWithMockPin(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM pins WHERE id = ?").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodDelete, "/pins/missing", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
