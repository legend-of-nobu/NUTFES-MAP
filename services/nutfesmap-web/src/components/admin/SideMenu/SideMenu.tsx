"use client";
import React, { useEffect } from "react";
import { PlanEditForm } from "./PlanEditForm/PlanEditForm";
import { AreaEditForm } from "./AreaEditForm/AreaEditForm";
import { MapEditForm, type MapEditFormProps } from "./MapEditForm/MapEditForm";

type SideMenuProps = {
  mode: "plan" | "area" | "map";
  onClose: () => void;
  // AdminPage から渡す（map が未選択の場合は undefined）
  mapEditProps?: MapEditFormProps;
};

export default function SideMenu({ mode, onClose, mapEditProps }: SideMenuProps) {
  // 「map」モードなのに props が無い = フォーム未初期化 → アラート表示して閉じる
  useEffect(() => {
    if (mode === "map" && !mapEditProps) {
      alert("編集対象のマップが未選択です。");
      onClose();
    }
  }, [mode, mapEditProps, onClose]);

  return (
    <div>
      {mode === "plan" && <PlanEditForm onClose={onClose} />}
      {mode === "area" && <AreaEditForm onClose={onClose} />}
      {mode === "map" && mapEditProps && (
        <MapEditForm {...mapEditProps} onClose={onClose} />
      )}
    </div>
  );
}
