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
  onAreaDeleted?: (pinId: string) => void;

  // 追加: 既存プランピン編集用
  editPlanPin?: ApiPin | null;
  onPlanUpdated?: (p: ApiPin) => void;
  onPlanDeleted?: (pinId: string) => void;
};

export default function SideMenu({
  mode,
  onClose,
  mapEditProps,
  pinContext,
  editAreaPin,
  onAreaUpdated,
  onAreaDeleted,
  editPlanPin,
  onPlanUpdated,
  onPlanDeleted,
}: SideMenuProps) {
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
          // 既存編集が指定されている場合は edit を渡す
          edit={
            editPlanPin && pinContext?.mapId
              ? {
                  mapId: pinContext.mapId,
                  pin: editPlanPin,
                  onUpdated: onPlanUpdated,
                  onDeleted: onPlanDeleted,
                }
              : undefined
          }
          // 既存編集が無ければ新規作成（placingフロー）のコンテキスト
          context={
            !editPlanPin
              ? {
                  mapId: pinContext?.mapId ?? null,
                  draftPos: pinContext?.draftPos ?? null,
                  onCreated: pinContext?.onPlanCreated,
                }
              : undefined
          }
        />
      )}

      {mode === "area" && (
        <AreaEditForm
          onClose={onClose}
          context={{
            mapId: pinContext?.mapId ?? null,
            draftPos: pinContext?.draftPos ?? null,
            onCreated: pinContext?.onAreaCreated,
          }}
          editPin={editAreaPin ?? undefined}
          onUpdated={onAreaUpdated}
          onDeleted={onAreaDeleted}
        />
      )}

      {mode === "map" && mapEditProps && <MapEditForm {...mapEditProps} onClose={onClose} />}
    </div>
  );
}
