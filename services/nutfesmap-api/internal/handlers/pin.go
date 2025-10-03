package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"nutfesmap/internal/repository"
)

// ====== ハンドラー ======

type PinHandler struct {
	Repo *repository.PinRepository
}

func NewPinHandler(r *repository.PinRepository) *PinHandler {
	return &PinHandler{Repo: r}
}

// GET /maps/:mapId/pins
// 指定マップに属するピン一覧を返す。modified_at DESC, id ASC（Repository側の並び）である。
func (h *PinHandler) ListByMap(c echo.Context) error {
	mapID := c.Param("mapId")
	if strings.TrimSpace(mapID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	ctx := c.Request().Context()
	items, err := h.Repo.FindByMapID(ctx, mapID)
	if err != nil {
		// map が存在しない場合は FK 的には区別が難しいため、空配列を返す方針でも良いが、
		// 仕様上は 200[] を返す。
		return err
	}

	// 名前昇順で安定化（UIでの視認性向上）。Repositoryの並びを尊重したい場合は削除可。
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	// ETag: mapID + 各要素(ID + ModifiedAt)
	hsh := sha256.New()
	hsh.Write([]byte(mapID))
	for _, p := range items {
		hsh.Write([]byte(p.ID))
		hsh.Write([]byte(p.ModifiedAt.UTC().Format(time.RFC3339Nano)))
	}
	c.Response().Header().Set("ETag", `W/"`+hex.EncodeToString(hsh.Sum(nil))+`"`)

	return c.JSON(http.StatusOK, items)
}

// POST /maps/:mapId/pins
// ピンを1件作成して返す。
func (h *PinHandler) CreateOnMap(c echo.Context) error {
	mapID := c.Param("mapId")
	if strings.TrimSpace(mapID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mapId is required")
	}

	var req repository.PinCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}

	newID := uuid.NewString()
	ctx := c.Request().Context()

	created, err := h.Repo.CreateOnMap(ctx, newID, mapID, &req)
	if err != nil {
		// 親マップ不在などのFKエラーを400に寄せる（詳細はエラーメッセージへ）
		// 必要なら MySQL のエラー番号で分岐して 404 に振り分けてもよい。
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if created == nil {
		// 直後の再読込失敗のフォールバック（まず起きない想定）
		now := time.Now().UTC()
		created = &repository.PinResponse{
			ID:          newID,
			MapID:       mapID,
			Name:        req.Name,
			Type:        strOr(req.Type, "exhibit"),
			LinkToMapID: req.LinkToMapID,
			XNorm:       req.XNorm,
			YNorm:       req.YNorm,
			Category:    req.Category,
			Status:      strOr(req.Status, "open"),
			WaitMinutes: intOr(req.WaitMinutes, 0),
			CreatedAt:   now,
			ModifiedAt:  now,
		}
	}

	// ETag（ID + ModifiedAt）
	setWeakETag(c, created.ID, created.ModifiedAt)

	return c.JSON(http.StatusCreated, created)
}

// GET /pins/:pinId
// ピン単体を返す。
func (h *PinHandler) Show(c echo.Context) error {
	pinID := c.Param("pinId")
	if strings.TrimSpace(pinID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pinId is required")
	}

	ctx := c.Request().Context()
	p, err := h.Repo.FindByID(ctx, pinID)
	if err != nil {
		return err
	}
	if p == nil {
		return echo.NewHTTPError(http.StatusNotFound, "pin not found")
	}

	// ETag（ID + ModifiedAt）
	setWeakETag(c, p.ID, p.ModifiedAt)

	return c.JSON(http.StatusOK, p)
}

// PATCH /pins/:pinId
// ピンの部分更新を行う。
func (h *PinHandler) Update(c echo.Context) error {
	pinID := c.Param("pinId")
	if strings.TrimSpace(pinID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pinId is required")
	}

	var req repository.PinUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body")
	}

	ctx := c.Request().Context()
	updated, err := h.Repo.UpdatePartial(ctx, pinID, &req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "pin not found")
		}
		// 検証エラーは400
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if updated == nil {
		return echo.NewHTTPError(http.StatusNotFound, "pin not found")
	}

	// ETag（ID + ModifiedAt）
	setWeakETag(c, updated.ID, updated.ModifiedAt)

	return c.JSON(http.StatusOK, updated)
}

// DELETE /pins/:pinId
// ピンを削除する。
func (h *PinHandler) Delete(c echo.Context) error {
	pinID := c.Param("pinId")
	if strings.TrimSpace(pinID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pinId is required")
	}

	ctx := c.Request().Context()
	if err := h.Repo.Delete(ctx, pinID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "pin not found")
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// ====== ユーティリティ ======

func setWeakETag(c echo.Context, id string, modAt time.Time) {
	h := sha256.New()
	h.Write([]byte(id))
	h.Write([]byte(modAt.UTC().Format(time.RFC3339Nano)))
	etag := `W/"` + hex.EncodeToString(h.Sum(nil)) + `"`
	c.Response().Header().Set("ETag", etag)
}

func strOr(p *string, def string) string {
	if p != nil && *p != "" {
		return *p
	}
	return def
}

func intOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}
