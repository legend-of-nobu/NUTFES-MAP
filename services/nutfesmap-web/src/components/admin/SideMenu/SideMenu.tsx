"use client";
import React, { useEffect } from "react";
import { PlanEditForm } from "./PlanEditForm/PlanEditForm";
import AreaEditForm from "./AreaEditForm/AreaEditForm";
import { MapEditForm, type MapEditFormProps } from "./MapEditForm/MapEditForm";
import type { ApiPin } from "@/components/map/PlanPin";
import type { ApiAreaPin } from "@/components/map/AreaPin";

type PinContext = {
  placingKind: "plan" | "area" | null;
  mapId: string | null; // 設置対象の親マップ
  draftPos: { xNorm: number; yNorm: number } | null;
  onAreaCreated?: (p: ApiAreaPin) => void;
  onPlanCreated?: (p: ApiPin) => void;
};

type SideMenuProps = {
  mode: "plan" | "area" | "map";
  onClose: () => void;
  // map 編集
  mapEditProps?: MapEditFormProps;
  // 新規ピン設置に必要な共通情報
  pinContext?: PinContext;
  // 既存エリアピン編集用
  editAreaPin?: { id: string; initialName: string } | null;
  onAreaUpdated?: (p: ApiAreaPin) => void;
};

export default function SideMenu({
  mode,
  onClose,
  mapEditProps,
  pinContext,
  editAreaPin,
  onAreaUpdated,
}: SideMenuProps) {
  // 「map」モードなのに props が無い = フォーム未初期化 → アラート表示して閉じる
  useEffect(() => {
    if (mode === "map" && !mapEditProps) {
      alert("編集対象のマップが未選択です。");
      onClose();
    }
  }, [mode, mapEditProps, onClose]);

  return (
    <div className="fixed top-0 right-0 bottom-0 w-[360px] bg-white shadow-2xl z-[1000] overflow-auto">
      {mode === "plan" && (
        <PlanEditForm
          onClose={onClose}
          context={{
            mapId: pinContext?.mapId ?? null,
            draftPos: pinContext?.draftPos ?? null,
            onCreated: pinContext?.onPlanCreated,
          }}
        />
      )}

      {mode === "area" && (
        <AreaEditForm
          onClose={onClose}
          // 新規作成（placing時）に使用
          context={{
            mapId: pinContext?.mapId ?? null,
            draftPos: pinContext?.draftPos ?? null,
            onCreated: pinContext?.onAreaCreated,
          }}
          // 既存編集（Editモードでピンをクリック時）に使用
          editPin={editAreaPin ?? undefined}
          onUpdated={onAreaUpdated}
        />
      )}

      {mode === "map" && mapEditProps && <MapEditForm {...mapEditProps} onClose={onClose} />}
    </div>
  );
}
