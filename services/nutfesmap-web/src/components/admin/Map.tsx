"use client";
import React, { useEffect, useRef, useState } from "react";
import MapImage from "@/components/map/MapImage";
import PlanPin, { ApiPin } from "@/components/map/PlanPin";
import type { SpotData } from "@/components/map/PlanPin";
import StairSelector from "@/components/map/StairSelector";
import AddPinButton from "@/components/map/AddPinButton";
import AreaPin, { ApiAreaPin } from "@/components/map/AreaPin";

type MapProps = {
  // 企画ピン
  pins?: ApiPin[];
  onPlanPinSelect?: (spot: SpotData) => void;

  // エリアピン
  areaPins?: ApiAreaPin[];
  onAreaPinSelect?: (area: ApiAreaPin) => void;

  // 「＋ピン」ボタン
  onAddPin?: () => void;

  // 設置モード：MapImage 内をクリックした座標を上位に返す
  onAddPinAt?: (xNorm: number, yNorm: number) => void;
  placing?: boolean;

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

  return (
    <div
      ref={hostRef}
      className={`relative h-full flex-1 bg-[#e2d7b5] overflow-hidden ${placing ? "cursor-crosshair" : ""}`}
    >
      <MapImage
        src={mapImageData ?? undefined}
        naturalWidth={naturalWidth}
        naturalHeight={naturalHeight}
        containerWidth={size.w}
        containerHeight={size.h}
        className="absolute inset-0"
        onAddPinAt={onAddPinAt} // ★ 設置モード中のみ AdminPage から渡される
      >
        {/* 企画ピン */}
        {pins.map((pin) => (
          <PlanPin key={pin.id} pin={pin} onSelect={onPlanPinSelect} />
        ))}

        {/* エリアピン */}
        {areaPins.map((area) => (
          <AreaPin key={area.id} area={area} onSelect={onAreaPinSelect} />
        ))}
      </MapImage>

      {/* 左下：階層セレクタ */}
      <div className="absolute bottom-4 left-4 z-10">
        <StairSelector floors={floors} selectedFloor={selectedFloor} onSelect={onSelectFloor} />
      </div>

      {/* 右下：「＋ピン」（edit時のみ） */}
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
