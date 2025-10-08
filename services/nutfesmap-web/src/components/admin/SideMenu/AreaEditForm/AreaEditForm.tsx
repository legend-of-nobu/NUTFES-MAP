"use client";
import React, { useEffect, useMemo, useState } from "react";
import { AreaNameForm } from "./AreaNameForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import type { ApiAreaPin } from "@/components/map/AreaPin";
import "../Style.css";

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

type CreateContext = {
  mapId: string | null;
  draftPos: { xNorm: number; yNorm: number } | null;
  onCreated?: (p: ApiAreaPin) => void;
};

type EditPin = {
  id: string;
  initialName: string;
};

type Props = {
  onClose: () => void;
  // 新規設置（既存機能）
  context?: CreateContext;
  // ★ 追加：既存ピンの編集
  editPin?: EditPin;
  // ★ 追加：編集反映
  onUpdated?: (p: ApiAreaPin) => void;
};

export const AreaEditForm: React.FC<Props> = ({ onClose, context, editPin, onUpdated }) => {
  const isEditing = !!editPin;
  const [areaName, setAreaName] = useState(isEditing ? editPin!.initialName : "");
  const [saving, setSaving] = useState(false);

  // 既存: 新規作成フローを使う状況では案内
  const canCreate = useMemo(
    () => !!context?.mapId && !!context?.draftPos && !isEditing,
    [context?.mapId, context?.draftPos, isEditing]
  );

  useEffect(() => {
    if (isEditing) setAreaName(editPin!.initialName);
  }, [isEditing, editPin]);

  // ★ 既存ピンの名称だけ更新（PATCH /pins/{pinId}）
  const patchExistingName = async () => {
    if (!isEditing) return;
    if (!areaName.trim()) {
      alert("エリア名を入力してください。");
      return;
    }
    try {
      setSaving(true);
      const res = await fetch(`${API}/pins/${encodeURIComponent(editPin!.id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name: areaName.trim() }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(`PATCH /pins/${editPin!.id} 失敗: ${res.status} ${text}`);
      }
      const p = await res.json();
      const updated: ApiAreaPin = {
        id: p.id,
        mapId: p.mapId,
        name: p.name,
        xNorm: p.xNorm,
        yNorm: p.yNorm,
        linkToMapId: p.linkToMapId ?? null,
      };
      onUpdated?.(updated);
      onClose();
    } catch (e) {
      console.error(e);
      alert("ピン名の更新に失敗しました。");
    } finally {
      setSaving(false);
    }
  };

  // 既存機能（新規作成）は残すが、今回の要件では呼ばれません
  const createNewAreaPin = async () => {
    if (!canCreate) return;
    if (!areaName.trim()) {
      alert("エリア名を入力してください。");
      return;
    }
    try {
      setSaving(true);

      // ここは従来仕様：まずマップ作成 → その mapId を linkToMapId に入れて親マップにピン作成
      // （今回の要件では使いません。必要時にエンドポイントへ置き換えてください）
      // --- ダミー（既存機能保持のための形だけ） ---
      alert("現在は新規作成フローを停止中です（名称変更のみ対応）。");
    } finally {
      setSaving(false);
    }
  };

  const onSave = async () => {
    if (isEditing) {
      await patchExistingName();
    } else {
      await createNewAreaPin();
    }
  };

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title">{isEditing ? "エリアピンを編集" : "ピンを編集"}</h2>

      <AreaNameForm value={areaName} onChange={setAreaName} />

      <div className="buttonRow">
        <SaveButton onClick={onSave} disabled={saving || !areaName.trim()} />
        <DeleteButton />
      </div>
    </div>
  );
};

export default AreaEditForm;
