"use client";
import React, { useEffect, useRef, useState } from "react";
import MapImage from "@/components/map/MapImage";
import PlanPin, { ApiPin, SpotData as PlanSpotData } from "@/components/map/PlanPin";
import StairSelector from "@/components/map/StairSelector";
import AddPinButton from "@/components/map/AddPinButton";
import AreaPin, { ApiAreaPin } from "@/components/map/AreaPin";

type PinKind = "plan" | "area";

type MapProps = {
  // 企画ピン
  pins?: ApiPin[];
  onPlanPinSelect?: (spot: PlanSpotData) => void;

  // エリアピン
  areaPins?: ApiAreaPin[];
  onAreaPinSelect?: (area: ApiAreaPin) => void;

  // 右下「＋ピン」
  onAddPin?: () => void;

  // 配置確定（MapImage内クリックで呼ばれる）
  onAddPinAt?: (xNorm: number, yNorm: number) => void;

  // 設置モード制御
  placing?: boolean;
  placingKind?: PinKind | null;

  // ★ 追加：クリック確定後の固定座標（ある間は追従を停止してゴースト固定）
  draftPos?: { xNorm: number; yNorm: number } | null;

  header?: React.ReactNode;
  mode?: "edit" | "user";
  floors?: string[];
  selectedFloor?: string;
  onSelectFloor?: (floor: string) => void;
  mapImageData?: string | null;
  mapId?: string | null;
  naturalWidth?: number;
  naturalHeight?: number;
};

export default function Map({
  pins = [],
  onPlanPinSelect = () => {},
  areaPins = [],
  onAreaPinSelect = () => {},
  onAddPin,
  onAddPinAt,
  placing = false,
  placingKind = null,
  draftPos = null, // ★ 追加
  header = null,
  mode = "user",
  floors = ["3F", "2F", "1F"],
  selectedFloor = "2F",
  onSelectFloor = () => {},
  mapImageData = null,
  mapId = null,
  naturalWidth = 0,
  naturalHeight = 0,
}: MapProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ w: 0, h: 0 });

  // ホバー中のゴースト座標（0..1）。クリック後は draftPos を使うので更新しない
  const [ghostPos, setGhostPos] = useState<{ x: number; y: number } | null>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const el = hostRef.current;
    const ro = new ResizeObserver((entries) => {
      const cr = entries[0].contentRect;
      setSize({ w: cr.width, h: cr.height });
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // 設置解除時はゴースト消去
  useEffect(() => {
    if (!placing) setGhostPos(null);
  }, [placing]);

  // === ゴーストの座標決定ロジック ===
  // 1) placing && !draftPos: カーソル追従（ghostPos を使用）
  // 2) placing && draftPos: クリック確定位置（draftPos を使用）
  const effectiveGhost =
    placing && placingKind
      ? draftPos
        ? { x: draftPos.xNorm, y: draftPos.yNorm }
        : ghostPos
      : null;

  // ゴースト ピンデータ（見た目だけ・pointer-events: none）
  const ghostPlan: ApiPin | null =
    effectiveGhost && placingKind === "plan"
      ? {
          id: "ghost-plan",
          mapId: mapId ?? "",
          name: "新規企画",
          description: null,
          descriptionImageData: null,
          type: "exhibit",
          linkToMapId: null,
          xNorm: effectiveGhost.x,
          yNorm: effectiveGhost.y,
          category: "plan",
          status: "open",
          waitMinutes: 0,
          createdAt: new Date().toISOString(),
          modifiedAt: new Date().toISOString(),
        }
      : null;

  const ghostArea: ApiAreaPin | null =
    effectiveGhost && placingKind === "area"
      ? {
          id: "ghost-area",
          mapId: mapId ?? "",
          name: "新規エリア",
          xNorm: effectiveGhost.x,
          yNorm: effectiveGhost.y,
          linkToMapId: null,
        }
      : null;

  return (
    <div ref={hostRef} className="relative h-full flex-1 bg-[#e2d7b5] overflow-hidden">
      <MapImage
        src={mapImageData ?? undefined}
        naturalWidth={naturalWidth}
        naturalHeight={naturalHeight}
        containerWidth={size.w}
        containerHeight={size.h}
        // クリックで確定（draftPos が既にある場合は Admin 側で無視される）
        onAddPinAt={onAddPinAt}
        // ホバー追従は draftPos が未確定の時だけ
        onHoverAt={(nx, ny) => {
          if (placing && !draftPos) setGhostPos({ x: nx, y: ny });
        }}
        // カーソル非表示は「クリック確定前のみ」
        placing={placing && !draftPos}
        className="absolute inset-0"
      >
        {/* 実ピン */}
        {pins.map((pin) => (
          <PlanPin key={pin.id} pin={pin} onSelect={onPlanPinSelect} />
        ))}

        {/* 実エリアピン */}
        {areaPins.map((area) => (
          <AreaPin key={area.id} area={area} onSelect={onAreaPinSelect} />
        ))}

        {/* ゴースト（固定 or 追従） */}
        {ghostPlan && <PlanPin pin={ghostPlan} ghost onSelect={() => {}} />}
        {ghostArea && <AreaPin area={ghostArea} ghost onSelect={() => {}} />}
      </MapImage>

      {/* 左下：階層セレクタ */}
      <div className="absolute bottom-4 left-4 z-10">
        <StairSelector floors={floors} selectedFloor={selectedFloor} onSelect={onSelectFloor} />
      </div>

      {/* 右下：ピン追加ボタン（edit時のみ） */}
      {mode === "edit" && (
        <div className="absolute bottom-6 right-6 z-10">
          <AddPinButton onClick={onAddPin} />
        </div>
      )}

      {/* 上部：AdminHeader */}
      <div className="absolute top-0 left-0 right-0 p-4 z-10">{header}</div>
    </div>
  );
}
