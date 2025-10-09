"use client";
import React, {
  useMemo,
  useRef,
  useEffect,
  useState,
  MouseEvent,
  useCallback,
  createContext,
  useContext,
} from "react";

type MapImageProps = {
  src: string | null | undefined;
  naturalWidth: number;
  naturalHeight: number;
  containerWidth: number;
  containerHeight: number;

  // 画像領域をクリックした時、正規化座標でコールバック（ピン設置）
  onAddPinAt?: (xNorm: number, yNorm: number) => void;

  // 設置モード中にカーソル正規化座標を通知（ゴースト追従用）
  onHoverAt?: (xNorm: number, yNorm: number) => void;

  // 設置モード中はカーソル非表示などの UI 切替に利用
  placing?: boolean;

  className?: string;
  style?: React.CSSProperties;

  // rectに一致する相対コンテナ内へ自由に children を描画（PlanPin/AreaPin/ゴースト等）
  children?: React.ReactNode;
};

type MapMetrics = {
  renderedWidth: number;
  renderedHeight: number;
  naturalWidth: number;
  naturalHeight: number;
  scale: number;
};

const defaultMetrics: MapMetrics = {
  renderedWidth: 0,
  renderedHeight: 0,
  naturalWidth: 1,
  naturalHeight: 1,
  scale: 1,
};

const MapMetricsContext = createContext<MapMetrics>(defaultMetrics);

export const useMapMetrics = () => useContext(MapMetricsContext);

function calcContainRect(
  containerW: number,
  containerH: number,
  imageW: number,
  imageH: number
) {
  if (containerW <= 0 || containerH <= 0 || imageW <= 0 || imageH <= 0)
    return { x: 0, y: 0, w: 0, h: 0 };
  const containerRatio = containerW / containerH;
  const imageRatio = imageW / imageH;
  let w = 0,
    h = 0;
  if (imageRatio > containerRatio) {
    w = containerW;
    h = w / imageRatio;
  } else {
    h = containerH;
    w = h * imageRatio;
  }
  const x = (containerW - w) / 2;
  const y = (containerH - h) / 2;
  return { x, y, w, h };
}

const MapImage: React.FC<MapImageProps> = ({
  src,
  naturalWidth,
  naturalHeight,
  containerWidth,
  containerHeight,
  onAddPinAt,
  onHoverAt,
  placing = false,
  className = "",
  style,
  children,
}) => {
  const viewportRef = useRef<HTMLDivElement>(null); // ビューポート（見える範囲）
  const worldRef = useRef<HTMLDivElement>(null); // 画像とピンを載せる層（拡大縮小・平行移動する）
  const [intrinsicSize, setIntrinsicSize] = useState<{ w: number; h: number } | null>(null);

  useEffect(() => {
    if (!src) {
      setIntrinsicSize(null);
      return;
    }

    let cancelled = false;
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      if (cancelled) return;
      const w = img.naturalWidth || 0;
      const h = img.naturalHeight || 0;
      if (w > 0 && h > 0) {
        setIntrinsicSize((prev) => (prev && prev.w === w && prev.h === h ? prev : { w, h }));
      } else {
        setIntrinsicSize(null);
      }
    };
    img.onerror = () => {
      if (!cancelled) setIntrinsicSize(null);
    };
    img.src = src;
    return () => {
      cancelled = true;
    };
  }, [src]);

  const effectiveNaturalWidth = intrinsicSize?.w && intrinsicSize.w > 0 ? intrinsicSize.w : naturalWidth;
  const effectiveNaturalHeight = intrinsicSize?.h && intrinsicSize.h > 0 ? intrinsicSize.h : naturalHeight;
  const safeNaturalWidth = effectiveNaturalWidth > 0 ? effectiveNaturalWidth : 1;
  const safeNaturalHeight = effectiveNaturalHeight > 0 ? effectiveNaturalHeight : 1;

  // 画像を「contain」した座標・サイズ
  const rect = useMemo(
    () => calcContainRect(containerWidth, containerHeight, safeNaturalWidth, safeNaturalHeight),
    [containerWidth, containerHeight, safeNaturalWidth, safeNaturalHeight]
  );

  const mapScale = rect.w > 0 ? rect.w / safeNaturalWidth : 1;
  const metrics: MapMetrics = useMemo(
    () => ({
      renderedWidth: rect.w,
      renderedHeight: rect.h,
      naturalWidth: safeNaturalWidth,
      naturalHeight: safeNaturalHeight,
      scale: mapScale,
    }),
    [rect.w, rect.h, safeNaturalWidth, safeNaturalHeight, mapScale]
  );

  // ====== ズーム・パンの状態 ======
  const MIN_SCALE = 1; // 等倍フィット
  const MAX_SCALE = 8;

  const [scale, setScale] = useState(1);
  const [tx, setTx] = useState(0); // translateX（px）
  const [ty, setTy] = useState(0); // translateY（px）

  // rect が更新されたらズーム・パンを初期化
  useEffect(() => {
    setScale(1);
    setTx(0);
    setTy(0);
  }, [rect.w, rect.h]);

  // ====== ユーティリティ ======
  // ビューポート内にできるだけ収まるようにパンをクランプ
  const clampPan = useCallback(
    (nextTx: number, nextTy: number, nextScale: number) => {
      const worldW = rect.w * nextScale;
      const worldH = rect.h * nextScale;

      const vpW = rect.w;
      const vpH = rect.h;

      let minX: number, maxX: number, minY: number, maxY: number;

      if (worldW <= vpW) {
        minX = maxX = (vpW - worldW) / 2;
      } else {
        minX = vpW - worldW;
        maxX = 0;
      }

      if (worldH <= vpH) {
        minY = maxY = (vpH - worldH) / 2;
      } else {
        minY = vpH - worldH;
        maxY = 0;
      }

      return {
        tx: Math.min(maxX, Math.max(minX, nextTx)),
        ty: Math.min(maxY, Math.max(minY, nextTy)),
      };
    },
    [rect.w, rect.h]
  );

  // カーソル位置を中心にズーム（ホイール・ピンチ共通ロジック）
  const zoomAt = useCallback(
    (clientX: number, clientY: number, zoomFactor: number) => {
      const el = viewportRef.current;
      if (!el) return;

      const vpBox = el.getBoundingClientRect();
      const vx = clientX - vpBox.left;
      const vy = clientY - vpBox.top;

      // world 座標（ズーム前）
      const wx = (vx - tx) / scale;
      const wy = (vy - ty) / scale;

      const nextScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale * zoomFactor));

      // ズーム後も同じ画素を同じ位置に
      const nextTx = vx - wx * nextScale;
      const nextTy = vy - wy * nextScale;

      const clamped = clampPan(nextTx, nextTy, nextScale);
      setScale(nextScale);
      setTx(clamped.tx);
      setTy(clamped.ty);
    },
    [scale, tx, ty, clampPan]
  );

  // ====== ホイールでズーム ======
  const onWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      const delta = e.deltaY;
      if (delta === 0) return;
      const factor = Math.exp(-delta * 0.001);
      zoomAt(e.clientX, e.clientY, factor);
      e.preventDefault();
    },
    [zoomAt]
  );

  // ====== ドラッグでパン（Pointer Events） ======
  const pointers = useRef<Map<number, { x: number; y: number }>>(new Map());
  const prevPinch = useRef<{ dist: number; cx: number; cy: number } | null>(null);

  const computeNormalizedAt = (clientX: number, clientY: number) => {
    const vp = viewportRef.current;
    if (!vp) return null;
    const vpBox = vp.getBoundingClientRect();
    const vx = clientX - vpBox.left;
    const vy = clientY - vpBox.top;
    const wx = (vx - tx) / scale;
    const wy = (vy - ty) / scale;
    const nx = wx / rect.w;
    const ny = wy / rect.h;
    const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
    return { nx: clamp01(nx), ny: clamp01(ny) };
  };

  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    (e.target as Element).setPointerCapture?.(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
  };

  const onPointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    pointers.current.delete(e.pointerId);
    if (pointers.current.size < 2) prevPinch.current = null;
  };

  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    // ★ まず常にホバー座標を通知（設置モードのゴースト追従用）
    if (onHoverAt) {
      const pos = computeNormalizedAt(e.clientX, e.clientY);
      if (pos) onHoverAt(pos.nx, pos.ny);
    }

    // 以下はパン/ピンチ操作（ドラッグ時のみ）
    const pts = pointers.current;
    if (!pts.has(e.pointerId)) return;

    const prev = pts.get(e.pointerId)!;
    const dx = e.clientX - prev.x;
    const dy = e.clientY - prev.y;
    pts.set(e.pointerId, { x: e.clientX, y: e.clientY });

    // 2本指 → ピンチズーム
    if (pts.size >= 2) {
      const [p1, p2] = Array.from(pts.values());
      const dist = Math.hypot(p2.x - p1.x, p2.y - p1.y);
      const cx = (p1.x + p2.x) / 2;
      const cy = (p1.y + p2.y) / 2;

      if (!prevPinch.current) {
        prevPinch.current = { dist, cx, cy };
        return;
      }

      const { dist: prevDist, cx: prevCx, cy: prevCy } = prevPinch.current;
      const factor = dist / prevDist || 1;
      const movedCx = cx - prevCx;
      const movedCy = cy - prevCy;

      let nextTx = tx + movedCx;
      let nextTy = ty + movedCy;

      const el = viewportRef.current;
      if (el) {
        const vpBox = el.getBoundingClientRect();
        const vx = cx - vpBox.left;
        const vy = cy - vpBox.top;

        const wx = (vx - nextTx) / scale;
        const wy = (vy - nextTy) / scale;

        const nextScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale * factor));
        nextTx = vx - wx * nextScale;
        nextTy = vy - wy * nextScale;

        const clamped = clampPan(nextTx, nextTy, nextScale);
        setScale(nextScale);
        setTx(clamped.tx);
        setTy(clamped.ty);
      }

      prevPinch.current = { dist, cx, cy };
      return;
    }

    // 1本指/マウス → パン
    const clamped = clampPan(tx + dx, ty + dy, scale);
    setTx(clamped.tx);
    setTy(clamped.ty);
  };

  // 画像領域クリック → 正規化座標で onAddPinAt
  const handleOverlayClick = (e: MouseEvent<HTMLDivElement>) => {
    if (!onAddPinAt) return;
    const pos = computeNormalizedAt(e.clientX, e.clientY);
    if (!pos) return;
    onAddPinAt(pos.nx, pos.ny);
  };

  const onDoubleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    zoomAt(e.clientX, e.clientY, 1.5);
  };

  return (
    <div
      className={`relative overflow-hidden select-none ${className}`}
      style={{
        width: containerWidth,
        height: containerHeight,
        backgroundColor: "#e2d7b5",
        ...style,
      }}
    >
      {/* rect で切り出したビューポート（この中だけをズーム・パン） */}
      <div
        ref={viewportRef}
        className="absolute touch-none"
        style={{
          left: rect.x,
          top: rect.y,
          width: rect.w,
          height: rect.h,
          cursor: placing ? "none" : "default", // ★ 設置中はカーソル非表示
        }}
        onWheel={onWheel}
        onPointerDown={onPointerDown}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
        onPointerMove={onPointerMove}
        onDoubleClick={onDoubleClick}
        onClick={handleOverlayClick}
      >
        {/* ズーム・パン対象の world レイヤ */}
        <div
          ref={worldRef}
          className="absolute"
          style={{
            width: rect.w,
            height: rect.h,
            transform: `translate(${tx}px, ${ty}px) scale(${scale})`,
            transformOrigin: "0 0",
          }}
        >
          {/* 背景画像（world の 0,0 にフィット） */}
          {src && rect.w > 0 && rect.h > 0 && (
            <img
              src={src}
              alt="map background"
              className="absolute pointer-events-none"
              style={{ left: 0, top: 0, width: rect.w, height: rect.h }}
              draggable={false}
            />
          )}

          {/* 相対(%基準)の子要素（ピン/ゴースト等） */}
          <MapMetricsContext.Provider value={metrics}>
            {children}
          </MapMetricsContext.Provider>
        </div>
      </div>
    </div>
  );
};

export default MapImage;
