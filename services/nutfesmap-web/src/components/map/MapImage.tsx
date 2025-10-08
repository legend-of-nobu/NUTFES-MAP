"use client";
import React, {
  useMemo,
  useRef,
  useEffect,
  useState,
  MouseEvent,
  useCallback,
} from "react";

export type PinType = { id: string | number; xNorm: number; yNorm: number };

type MapImageProps = {
  src: string | null | undefined;
  naturalWidth: number;
  naturalHeight: number;
  containerWidth: number;
  containerHeight: number;

  // 既存（px直指定ピン）を使うならこの2つは適宜利用してください
  pins?: PinType[];
  onPinClick?: (p: PinType) => void;

  // 画像領域をクリックした時、正規化座標でコールバック
  onAddPinAt?: (xNorm: number, yNorm: number) => void;

  // ★ ゴーストピン追従のために、ポインタ位置の正規化座標を通知
  onHoverAt?: (xNorm: number, yNorm: number) => void;
  onHoverLeave?: () => void;

  className?: string;
  style?: React.CSSProperties;

  // ★ rectに一致する相対コンテナ内へ自由に children を描画（PlanPin/AreaPin/ゴースト）
  children?: React.ReactNode;
};

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
  pins = [],
  onPinClick,
  onAddPinAt,
  onHoverAt,
  onHoverLeave,
  className = "",
  style,
  children,
}) => {
  const viewportRef = useRef<HTMLDivElement>(null); // ビューポート（見える範囲）
  const worldRef = useRef<HTMLDivElement>(null); // 画像とピンを載せる層（拡大縮小・平行移動する）

  // 画像を「contain」した座標・サイズ
  const rect = useMemo(
    () => calcContainRect(containerWidth, containerHeight, naturalWidth, naturalHeight),
    [containerWidth, containerHeight, naturalWidth, naturalHeight]
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
      // world の表示サイズ（scale後）
      const worldW = rect.w * nextScale;
      const worldH = rect.h * nextScale;

      // ビューポートサイズ
      const vpW = rect.w;
      const vpH = rect.h;

      // world は viewport 内に「少なくとも」覆い被さってほしい（白地を見せない）
      // world が viewport より小さい場合は中央寄せ
      let minX: number, maxX: number, minY: number, maxY: number;

      if (worldW <= vpW) {
        // 横方向は中央寄せ、パン無効
        minX = maxX = (vpW - worldW) / 2;
      } else {
        // world が広いときは、端がちょうど見切れる位置までを許可
        minX = vpW - worldW;
        maxX = 0;
      }

      if (worldH <= vpH) {
        // 縦方向も同様
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

      // ビューポート内座標
      const vx = clientX - vpBox.left;
      const vy = clientY - vpBox.top;

      // world 座標（ズーム前）
      const wx = (vx - tx) / scale;
      const wy = (vy - ty) / scale;

      // 新しいスケールを計算
      const nextScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale * zoomFactor));

      // クライアント座標 (vx, vy) にある world 点 (wx, wy) が、ズーム後も同じ位置に見えるように平行移動を再計算
      let nextTx = vx - wx * nextScale;
      let nextTy = vy - wy * nextScale;

      // クランプ
      const clamped = clampPan(nextTx, nextTy, nextScale);

      setScale(nextScale);
      setTx(clamped.tx);
      setTy(clamped.ty);
    },
    [scale, tx, ty, clampPan]
  );

  // ====== ホイールでズーム（トラックパッドにも自然に） ======
  const onWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      const delta = e.deltaY;
      if (delta === 0) return;
      const factor = Math.exp(-delta * 0.001); // 下回しで拡大，上回しで縮小
      zoomAt(e.clientX, e.clientY, factor);
      e.preventDefault();
    },
    [zoomAt]
  );

  // ★ ポインタのクライアント座標 → 正規化座標(0..1)に変換して onHoverAt に通知
  const notifyHover = useCallback(
    (clientX: number, clientY: number) => {
      if (!onHoverAt) return;
      const vp = viewportRef.current;
      if (!vp) return;

      const vpBox = vp.getBoundingClientRect();
      const vx = clientX - vpBox.left;
      const vy = clientY - vpBox.top;

      // world 座標（scale/translate を逆変換）
      const wx = (vx - tx) / scale;
      const wy = (vy - ty) / scale;

      // 正規化 0..1
      const nx = wx / rect.w;
      const ny = wy / rect.h;

      const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
      onHoverAt(clamp01(nx), clamp01(ny));
    },
    [onHoverAt, rect.w, rect.h, scale, tx, ty]
  );

  // ====== ドラッグでパン（Pointer Events） & ホバー通知 ======
  const pointers = useRef<Map<number, { x: number; y: number }>>(new Map());
  const prevPinch = useRef<{ dist: number; cx: number; cy: number } | null>(null);

  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    (e.target as Element).setPointerCapture?.(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    notifyHover(e.clientX, e.clientY);
  };

  const onPointerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    pointers.current.delete(e.pointerId);
    if (pointers.current.size < 2) prevPinch.current = null;
    notifyHover(e.clientX, e.clientY);
  };

  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const pts = pointers.current;
    if (pts.has(e.pointerId)) {
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
          notifyHover(e.clientX, e.clientY);
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
      } else {
        // 1本指/マウス → パン
        const clamped = clampPan(tx + dx, ty + dy, scale);
        setTx(clamped.tx);
        setTy(clamped.ty);
      }
    }

    // ホバー座標の通知（ドラッグ中/非ドラッグ中どちらでも）
    notifyHover(e.clientX, e.clientY);
  };

  const onPointerLeave = () => {
    onHoverLeave?.();
  };

  // ====== 画像領域クリック → 正規化座標で onAddPinAt ======
  const handleOverlayClick = (e: MouseEvent<HTMLDivElement>) => {
    if (!onAddPinAt) return;

    const vp = viewportRef.current;
    if (!vp) return;

    const vpBox = vp.getBoundingClientRect();
    const vx = e.clientX - vpBox.left;
    const vy = e.clientY - vpBox.top;

    // world 座標（scale/translate を逆変換）
    const wx = (vx - tx) / scale;
    const wy = (vy - ty) / scale;

    // 正規化 0..1
    const nx = wx / rect.w;
    const ny = wy / rect.h;

    const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
    onAddPinAt(clamp01(nx), clamp01(ny));
  };

  // ====== ダブルクリックでサクッと拡大 ======
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
        style={{ left: rect.x, top: rect.y, width: rect.w, height: rect.h }}
        onWheel={onWheel}
        onPointerDown={onPointerDown}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
        onPointerMove={onPointerMove}
        onPointerLeave={onPointerLeave}
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

          {/* ここに % 基準の PlanPin / AreaPin / Ghost を置く */}
          {children}
        </div>
      </div>

      {/* 既存の px 直打ちピンを使うなら、必要に応じてここで rect を使って px 変換して配置してください */}
    </div>
  );
};

export default MapImage;
