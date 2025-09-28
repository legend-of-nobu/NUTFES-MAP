package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"nutfesmap/internal/handlers"
	"nutfesmap/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"

	"github.com/go-playground/validator/v10"
)

// EchoのValidator実装
type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

// Echo + Handler + sqlmock をまとめて起動
func setupEchoWithMock(t *testing.T) (*echo.Echo, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	mapRepo := repository.NewMapRepository(db)
	mapH := handlers.NewMapHandler(mapRepo)

	e := echo.New()
	e.Validator = &customValidator{v: validator.New()}

	// ルーティング（本番と同じパス）
	e.POST("/maps", mapH.Create)
	e.GET("/maps/index", mapH.Index)

	cleanup := func() {
		_ = db.Close()
	}
	return e, mock, cleanup
}

func TestMapHandler_Create_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// --- 期待されるSQL（Insert → main select → count → children select） ---
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			sqlmock.AnyArg(), // id ("map_"+uuid) → ここはハンドラー内で生成済みの値を直接知らないので AnyArg
			"キャンパスマップ2025",
			sqlmock.AnyArg(), // image_data(base64)
			4096,
			3072,
			nil,              // parent_map_id
			true,             // has_floors
			3,                // floor_count
			sqlmock.AnyArg(), // created_at (time.Now().UTC())
			sqlmock.AnyArg(), // modified_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// main select
	now := time.Now().UTC()
	mainCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	mainRow := sqlmock.NewRows(mainCols).AddRow(
		"map_dummy", "キャンパスマップ2025", "iVBORw0K...", 4096, 3072,
		nil, true, 3, now, now, nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE id = ? AND deleted_at IS NULL
		 LIMIT 1
	`)).
		WithArgs(sqlmock.AnyArg()). // ハンドラーで生成した newID
		WillReturnRows(mainRow)

	// children count
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*) FROM maps WHERE parent_map_id = ? AND deleted_at IS NULL
	`)).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// children list
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count
		  FROM maps
		 WHERE parent_map_id = ? AND deleted_at IS NULL
		 ORDER BY name
	`)).
		WithArgs("map_dummy").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "has_floors", "floor_count"})) // 0行

	// --- リクエスト作成 ---
	reqBody := handlers.MapCreateRequest{
		Name:          "キャンパスマップ2025",
		ImageData:     "iVBORw0K...",
		NaturalWidth:  4096,
		NaturalHeight: 3072,
		HasFloors:     true,
		FloorCount:    3,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// --- 実行 ---
	e.ServeHTTP(rec, req)

	// --- 検証 ---
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}

	// レスポンスの最小検証（構造が正しいこと）
	var resp handlers.MapResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if resp.Name != reqBody.Name || resp.NaturalWidth != 4096 || resp.ChildrenCount != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Create_ValidationError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// name が無い → バリデーション 400
	reqBody := map[string]any{
		"imageData":     "iVBORw0K...",
		"naturalWidth":  1024,
		"naturalHeight": 768,
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	// SQL は一切呼ばれない想定
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQLs were executed: %v", err)
	}
}

func TestMapHandler_Create_InsertError(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// Insert がエラーを返すパス
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO maps (
			id, name, image_data, natural_width, natural_height,
			parent_map_id, has_floors, floor_count, created_at, modified_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)
	`)).
		WithArgs(
			sqlmock.AnyArg(),
			"ERR-MAP",
			sqlmock.AnyArg(),
			100, 100,
			nil,
			false, 0,
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(assertErr("insert failed"))

	reqBody := handlers.MapCreateRequest{
		Name:          "ERR-MAP",
		ImageData:     "AAAA",
		NaturalWidth:  100,
		NaturalHeight: 100,
		HasFloors:     false,
		FloorCount:    0,
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/maps", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Index_OK(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	now := time.Now().UTC()

	// 1) 親一覧（2件）
	parentCols := []string{
		"id", "name", "image_data", "natural_width", "natural_height",
		"parent_map_id", "has_floors", "floor_count", "created_at", "modified_at", "deleted_at",
	}
	// created_at DESC を想定して p1 → p2 の順で返す
	parentRows := sqlmock.NewRows(parentCols).
		AddRow("p1", "Campus A", "BASE64A", 4096, 3072, nil, true, 3, now, now, nil).
		AddRow("p2", "Campus B", "BASE64B", 2048, 1536, nil, false, 0, now.Add(-time.Minute), now.Add(-time.Minute), nil)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnRows(parentRows)

	// 2) 子件数の集約（IN (?,?)）
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT parent_map_id, COUNT(*)
		  FROM maps
		 WHERE deleted_at IS NULL
		   AND parent_map_id IN (?,?)
		 GROUP BY parent_map_id
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(
			sqlmock.NewRows([]string{"parent_map_id", "count"}).
				AddRow("p1", 2).
				AddRow("p2", 1),
		)

	// 3) 子の軽量一覧（IN (?,?)、名前昇順）
	childCols := []string{"id", "name", "has_floors", "floor_count", "parent_map_id"}
	childRows := sqlmock.NewRows(childCols).
		AddRow("c11", "1F", false, 0, "p1").
		AddRow("c12", "2F", false, 0, "p1").
		AddRow("c21", "展示エリア", false, 0, "p2")

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, has_floors, floor_count, parent_map_id
		  FROM maps
		 WHERE deleted_at IS NULL
		   AND parent_map_id IN (?,?)
		 ORDER BY name ASC
	`)).
		WithArgs("p1", "p2").
		WillReturnRows(childRows)

	// --- 実行 ---
	req := httptest.NewRequest(http.MethodGet, "/maps/index", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- 検証 ---
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if etag := rec.Header().Get("ETag"); etag == "" {
		t.Fatalf("ETag header must be set")
	}

	// レスポンスの最小検証
	var list struct {
		Items []handlers.MapResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("json unmarshal error: %v, body=%s", err, rec.Body.String())
	}
	if len(list.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(list.Items))
	}
	// p1
	if list.Items[0].ID != "p1" || list.Items[0].ChildrenCount != 2 || len(list.Items[0].Children) != 2 {
		t.Fatalf("unexpected p1 aggregate: %+v", list.Items[0])
	}
	// 子は名前昇順（"1F","2F"）
	if list.Items[0].Children[0].Name != "1F" || list.Items[0].Children[1].Name != "2F" {
		t.Fatalf("children not sorted by name asc: %+v", list.Items[0].Children)
	}
	// p2
	if list.Items[1].ID != "p2" || list.Items[1].ChildrenCount != 1 || len(list.Items[1].Children) != 1 {
		t.Fatalf("unexpected p2 aggregate: %+v", list.Items[1])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestMapHandler_Index_SQL_Error(t *testing.T) {
	e, mock, cleanup := setupEchoWithMock(t)
	defer cleanup()

	// 親の最初のSELECTでエラーを返す
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, image_data, natural_width, natural_height,
		       parent_map_id, has_floors, floor_count, created_at, modified_at, deleted_at
		  FROM maps
		 WHERE parent_map_id IS NULL
		   AND deleted_at IS NULL
		 ORDER BY created_at DESC
	`)).WillReturnError(assertErr("parent select failed"))

	req := httptest.NewRequest(http.MethodGet, "/maps/index", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 固定エラータイプ（比較しやすいように）
type assertErr string

func (e assertErr) Error() string { return string(e) }
