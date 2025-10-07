"use client";
import React from "react";
import Pin from "@/components/map/Pin";
import StairSelector from "@/components/map/StairSelector";
import AddPinButton from "@/components/map/AddPinButton";

type PinType = {
  id: string | number;
  xNorm: number;
  yNorm: number;
  // 必要ならメタ情報をここに追加（UIは変えない）
};

type MapProps = {
  // 既存
  pins?: PinType[];
  onPinClick?: (p: PinType) => void;
  onAddPin?: () => void;
  header?: React.ReactNode;
  mode?: "edit" | "user";

  // セレクタ系（見た目は既存のまま）
  floors?: string[];
  selectedFloor?: string;
  onSelectFloor?: (floor: string) => void;

  // ★AdminPageから渡される“背景画像”と“mapId”（mapIdはts的に受けるだけ）
  mapImageData?: string | null; // dataURL（toPreviewUrl済み）
  mapId?: string | null;
};

export default function Map({
  pins = [],
  onPinClick = () => {},
  onAddPin = () => {},
  header = null,
  mode = "user",
  floors = ["3F", "2F", "1F"],
  selectedFloor = "2F",
  onSelectFloor = () => {},
  mapImageData = null,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  mapId = null,
}: MapProps) {
  return (
    <div className="relative flex-1 bg-[#e2d7b5] overflow-hidden">
      {/* 背景マップ画像（画像があれば全面に表示） */}
      {mapImageData && (
        <img
          src={mapImageData}
          alt="map background"
          className="absolute inset-0 w-full h-full object-contain select-none pointer-events-none"
          style={{ zIndex: 0 }} // 背景は最背面・クリック不可
        />
      )}

      {/* ピン配置（既存の見た目を維持） */}
      <div className="absolute inset-0" style={{ zIndex: 1 }}>
        {pins.map((p) => (
          <Pin key={p.id} pin={p} onClick={() => onPinClick(p)} />
        ))}
      </div>

      {/* 左下：階層セレクタ（見た目は同一） */}
      <div className="absolute bottom-4 left-4" style={{ zIndex: 2 }}>
        <StairSelector
          floors={floors}
          selectedFloor={selectedFloor}
          onSelect={onSelectFloor}
        />
      </div>

      {/* 右下：ピン追加ボタン（edit時のみ） */}
      {mode === "edit" && (
        <div className="absolute bottom-6 right-6" style={{ zIndex: 2 }}>
          <AddPinButton onClick={onAddPin} />
        </div>
      )}

      {/* 上部：AdminHeader */}
      <div className="absolute top-0 left-0 right-0 p-4" style={{ zIndex: 3 }}>
        {header}
      </div>
    </div>
  );
}
