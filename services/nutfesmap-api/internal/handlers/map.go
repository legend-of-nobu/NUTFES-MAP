package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"nutfesmap/internal/repository"
)

// ---- Request DTO ----

// POST /maps（空マップ作成）リクエスト
type MapCreateRequest struct {
	ParentMapID *string `json:"parentMapId"`
}

// レスポンス全体ラッパ（/maps/index）
type MapsIndexResponse struct {
	Items []repository.MapResponse `json:"items"`
}

type MapHandler struct {
	Repo *repository.MapRepository
}

func NewMapHandler(r *repository.MapRepository) *MapHandler {
	return &MapHandler{Repo: r}
}

// POST /maps
// 空のマップのみ作成。name / imageData / natural* / floors は受け付けない。
func (h *MapHandler) Create(c echo.Context) error {
	var req MapCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
	}

	newID := uuid.NewString()
	ctx := c.Request().Context()

	// 空マップを作成（name/image/natural_* は未設定、親のみ任意）
	if err := h.Repo.CreateEmptyMapTx(ctx, newID, req.ParentMapID); err != nil {
		// DB起因の失敗はそのまま Echo が 500 にマップ
		return err
	}

	// 作成直後の状態を取得（常に repository.MapResponse を返す）
	res, err := h.Repo.FindMapResponseByID(ctx, newID)
	if err != nil {
		return err
	}
	if res == nil {
		// まれに直後の再読込に失敗した場合のフォールバック
		now := time.Now().UTC()
		res = &repository.MapResponse{
			ID:            newID,
			Name:          "",
			ImageData:     "",
			NaturalWidth:  0,
			NaturalHeight: 0,
			ParentMapID:   req.ParentMapID,
			HasFloors:     false,
			FloorCount:    0,
			ChildrenCount: 0,
			Children:      []repository.MapChildRefDTO{},
			CreatedAt:     now,
			ModifiedAt:    now,
		}
	}

	// ETag: ID + ModifiedAt
	hash := sha256.New()
	hash.Write([]byte(res.ID))
	hash.Write([]byte(res.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusCreated, res)
}

// GET /maps/index
func (h *MapHandler) Index(c echo.Context) error {
	ctx := c.Request().Context()

	ags, err := h.Repo.FindIndexAggregates(ctx)
	if err != nil {
		return err
	}

	items := make([]repository.MapResponse, 0, len(ags))
	hash := sha256.New()

	for _, ag := range ags {
		// 子を Name 昇順で安定化
		sort.Slice(ag.Children, func(i, j int) bool {
			return ag.Children[i].Name < ag.Children[j].Name
		})

		children := make([]repository.MapChildRefDTO, 0, len(ag.Children))
		for _, ch := range ag.Children {
			children = append(children, repository.MapChildRefDTO{
				ID:         ch.ID,
				Name:       ch.Name,
				HasFloors:  ch.HasFloors,
				FloorCount: ch.FloorCount,
			})
		}

		base := ag.Base
		items = append(items, repository.MapResponse{
			ID:            base.ID,
			Name:          base.Name,
			ImageData:     base.ImageData, // LONGTEXT(base64)
			NaturalWidth:  base.NaturalWidth,
			NaturalHeight: base.NaturalHeight,
			ParentMapID:   base.ParentMapID,
			HasFloors:     base.HasFloors,
			FloorCount:    base.FloorCount,
			ChildrenCount: ag.ChildrenCount,
			Children:      children,
			CreatedAt:     base.CreatedAt,
			ModifiedAt:    base.ModifiedAt,
		})

		// ETag 材料（ID+ModifiedAt）
		hash.Write([]byte(base.ID))
		hash.Write([]byte(base.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	}

	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, MapsIndexResponse{Items: items})
}

// GET /maps/:mapId （地図メタ取得）
func (h *MapHandler) Show(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	res, err := h.Repo.FindMapResponseByID(ctx, mapID)
	if err != nil {
		return err
	}
	if res == nil {
		return echo.NewHTTPError(http.StatusNotFound, "map not found")
	}

	// 子は名前昇順で安定化
	sort.Slice(res.Children, func(i, j int) bool {
		return res.Children[i].Name < res.Children[j].Name
	})

	// 単体ETag: ID + ModifiedAt
	hash := sha256.New()
	hash.Write([]byte(res.ID))
	hash.Write([]byte(res.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, res)
}

// PATCH /maps/:mapId
func (h *MapHandler) Update(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	// 受け取り: name / imageData / naturalWidth / naturalHeight / parentMapId のみ
	var req repository.MapUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}

	ctx := c.Request().Context()
	updated, err := h.Repo.UpdatePartial(ctx, mapID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "map not found")
		}
		// ドメイン検証エラーなどは 400 に寄せる（UpdatePartial は検証時に error を返す）
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if updated == nil {
		return echo.NewHTTPError(http.StatusNotFound, "map not found")
	}

	// 子は名前昇順で安定化
	sort.Slice(updated.Children, func(i, j int) bool {
		return updated.Children[i].Name < updated.Children[j].Name
	})

	// ETag: ID + ModifiedAt
	hash := sha256.New()
	hash.Write([]byte(updated.ID))
	hash.Write([]byte(updated.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)

	return c.JSON(http.StatusOK, updated)
}

// DELETE /maps/:mapId
func (h *MapHandler) Delete(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	_, _, err := h.Repo.DeleteCascade(ctx, mapID)
	if err != nil {
		// 対象なし
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "map not found")
		}
		// それ以外は内部エラーとして扱う（Echo 側で 500 にマップ）
		return err
	}

	// 本体・子マップ・ピンを再帰的に削除済み
	return c.NoContent(http.StatusNoContent)
}

// POST /maps/:mapId/floors
// 指定 root の直下に空の Floor を1件追加し、作成された Floor の MapResponse を返します。
func (h *MapHandler) CreateFloor(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	newID := uuid.NewString()
	ctx := c.Request().Context()

	// 親(root)存在チェック＋作成（内部で FOR UPDATE / 集約更新あり）
	if err := h.Repo.CreateEmptyMapTx(ctx, newID, &mapID); err != nil {
		// 親なし → 404 に寄せる（Repository は "parent root not found" のエラー文字列）
		if strings.Contains(err.Error(), "parent root not found") {
			return echo.NewHTTPError(http.StatusNotFound, "parent root not found")
		}
		return err // その他は 500
	}

	// 作成した Floor を返却
	res, err := h.Repo.FindMapResponseByID(ctx, newID)
	if err != nil {
		return err
	}
	if res == nil {
		// 直後の再取得に失敗した場合のフォールバック
		now := time.Now().UTC()
		res = &repository.MapResponse{
			ID:            newID,
			Name:          "",
			ImageData:     "",
			NaturalWidth:  0,
			NaturalHeight: 0,
			ParentMapID:   &mapID,
			HasFloors:     false,
			FloorCount:    0,
			ChildrenCount: 0,
			Children:      []repository.MapChildRefDTO{},
			CreatedAt:     now,
			ModifiedAt:    now,
		}
	}

	// ETag（ID + ModifiedAt）
	hsh := sha256.New()
	hsh.Write([]byte(res.ID))
	hsh.Write([]byte(res.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	c.Response().Header().Set("ETag", `W/"`+hex.EncodeToString(hsh.Sum(nil))+`"`)

	return c.JSON(http.StatusCreated, res)
}

// DELETE /maps/:mapId/floors/:floorIndex
// 最上階のみ削除可能。floorIndex は 1..floor_count で、現状の floor_count と一致したときのみ削除。
func (h *MapHandler) DeleteTopFloor(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}
	idxStr := c.Param("floorIndex")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "floorIndex must be an integer")
	}

	ctx := c.Request().Context()
	if err := h.Repo.DeleteTopFloorByIndex(ctx, mapID, idx); err != nil {
		// NOT FOUND 相当
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "map not found")
		}
		// ドメイン検証エラー系は 400 に寄せる
		if strings.Contains(err.Error(), "floorIndex must be") ||
			strings.Contains(err.Error(), "no floors to delete") ||
			strings.Contains(err.Error(), "only top floor") {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		// その他は 500
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// GET /maps/:mapId/floors
// 指定IDが root でも floor でも受け取り、所属 root の全フロアを 1F.. の順で返します。
func (h *MapHandler) ListFloors(c echo.Context) error {
	mapID := c.Param("mapId")
	if mapID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	fs, err := h.Repo.FindFloorStackByAnyID(ctx, mapID)
	if err != nil {
		return err
	}
	if fs == nil {
		return echo.NewHTTPError(http.StatusNotFound, "map not found")
	}

	// ETag（rootID + 各階 ModifiedAt）
	hsh := sha256.New()
	hsh.Write([]byte(fs.RootMapID))
	for _, it := range fs.Items {
		hsh.Write([]byte(it.Map.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	}
	c.Response().Header().Set("ETag", `W/"`+hex.EncodeToString(hsh.Sum(nil))+`"`)

	// repository.FloorStackResponse をそのまま返却
	return c.JSON(http.StatusOK, fs)
}
