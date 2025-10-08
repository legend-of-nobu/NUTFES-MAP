"use client";
import React, { useEffect } from "react";
import { PlanEditForm } from "./PlanEditForm/PlanEditForm";
import { AreaEditForm } from "./AreaEditForm/AreaEditForm";
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
  // ★ 新規ピン設置に必要な共通情報（既存機能）
  pinContext?: PinContext;
  // ★ 追加：既存エリアピンの編集ターゲット
  areaEditTarget?: ApiAreaPin | null;
  // ★ 追加：名称更新の反映
  onAreaUpdated?: (p: ApiAreaPin) => void;
};

export default function SideMenu({
  mode,
  onClose,
  mapEditProps,
  pinContext,
  areaEditTarget,
  onAreaUpdated,
}: SideMenuProps) {
  // 「map」モードなのに props が無い = フォーム未初期化 → アラート表示して閉じる
  useEffect(() => {
    if (mode === "map" && !mapEditProps) {
      alert("編集対象のマップが未選択です。");
      onClose();
    }
  }, [mode, mapEditProps, onClose]);

  // レイヤは常に最前面
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
          // ★ 既存ピン編集か？（editPin があれば編集。なければ既存の新規設置フロー）
          editPin={
            areaEditTarget
              ? { id: areaEditTarget.id, initialName: areaEditTarget.name }
              : undefined
          }
          onUpdated={onAreaUpdated}
          context={
            !areaEditTarget
              ? {
                  mapId: pinContext?.mapId ?? null,
                  draftPos: pinContext?.draftPos ?? null,
                  onCreated: pinContext?.onAreaCreated,
                }
              : undefined
          }
        />
      )}

      {mode === "map" && mapEditProps && <MapEditForm {...mapEditProps} onClose={onClose} />}
    </div>
  );
}
