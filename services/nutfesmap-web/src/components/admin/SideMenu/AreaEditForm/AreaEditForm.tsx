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
  mapId: string | null; // 親マップ（ピンを置く側）
  draftPos: { xNorm: number; yNorm: number } | null; // クリック確定座標
  onCreated?: (p: ApiAreaPin) => void; // 作成結果を親へ
};

type EditPin = {
  id: string;          // 既存ピンID
  initialName: string; // 既存名
};

type Props = {
  onClose: () => void;
  /** 新規作成フロー（ピン設置時だけ渡ってくる） */
  context?: CreateContext;
  /** 既存ピン編集（Editモードでピンをクリック時だけ渡ってくる） */
  editPin?: EditPin;
  /** 既存ピン更新後に親へ反映（名称変更） */
  onUpdated?: (p: ApiAreaPin) => void;
};

export const AreaEditForm: React.FC<Props> = ({ onClose, context, editPin, onUpdated }) => {
  const isEditing = !!editPin;
  const [areaName, setAreaName] = useState(isEditing ? editPin!.initialName : "");
  const [saving, setSaving] = useState(false);

  const canCreate = useMemo(
    () => !!context?.mapId && !!context?.draftPos && !isEditing,
    [context?.mapId, context?.draftPos, isEditing]
  );

  useEffect(() => {
    if (isEditing) setAreaName(editPin!.initialName);
  }, [isEditing, editPin]);

  // 新規作成
  const createAreaPin = async () => {
    if (!canCreate) return;
    const parentMapId = context!.mapId!;
    const { xNorm, yNorm } = context!.draftPos!;
    try {
      setSaving(true);

      // 1) 遷移先用の空マップを作成
      const mapRes = await fetch(`${API}/maps`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name: areaName || "", parentMapId: parentMapId }),
      });
      if (!mapRes.ok) {
        const t = await mapRes.text();
        throw new Error(`POST /maps 失敗: ${mapRes.status} ${t}`);
      }
      const createdMap = await mapRes.json(); // { id, ... }
      const linkToMapId: string = createdMap.id;

      // 2) 親マップにエリアピンを作成（※複数形 /pins）
      const pinRes = await fetch(`${API}/maps/${encodeURIComponent(parentMapId)}/pins`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          name: areaName || "エリア",
          type: "area_selector",
          linkToMapId,
          xNorm,
          yNorm,
          // サーバのDTOでは category は必須なので既存と整合する値を送る
          category: "plan",
          status: "open",
          waitMinutes: 0,
        }),
      });
      if (!pinRes.ok) {
        const t = await pinRes.text();
        throw new Error(`POST /maps/${parentMapId}/pins 失敗: ${pinRes.status} ${t}`);
      }
      const createdPin = await pinRes.json();

      const newArea: ApiAreaPin = {
        id: createdPin.id,
        mapId: createdPin.mapId,
        name: createdPin.name,
        xNorm: createdPin.xNorm,
        yNorm: createdPin.yNorm,
        linkToMapId: createdPin.linkToMapId ?? linkToMapId,
      };

      context?.onCreated?.(newArea);
      onClose();
    } catch (e) {
      console.error(e);
      alert("エリアピンの作成に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  // 既存ピン名の更新（PATCH）
  const patchAreaPinName = async () => {
    if (!isEditing) return;
    try {
      setSaving(true);
      const res = await fetch(`${API}/pins/${encodeURIComponent(editPin!.id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ name: areaName }),
      });
      if (!res.ok) {
        const t = await res.text();
        throw new Error(`PATCH /pins/${editPin!.id} 失敗: ${res.status} ${t}`);
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
      alert("エリアピンの更新に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  const handleSave = () => {
    if (isEditing) return void patchAreaPinName();
    if (canCreate) return void createAreaPin();
    alert("必要な情報が不足しています。");
  };

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title">{isEditing ? "エリアピンを編集" : "エリアピンを作成"}</h2>

      <AreaNameForm value={areaName} onChange={setAreaName} />

      <div className="buttonRow">
        <SaveButton onClick={handleSave} disabled={saving} />
        <DeleteButton />
      </div>
    </div>
  );
};

export default AreaEditForm;
