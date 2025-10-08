"use client";
import React, { useMemo, useState } from "react";
import { AreaNameForm } from "./AreaNameForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import "../Style.css";

import type { ApiAreaPin } from "@/components/map/AreaPin";

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

type Props = {
  onClose: () => void;
  context?: {
    mapId: string | null;                           // 親マップ（ピンを立てる側）
    draftPos: { xNorm: number; yNorm: number } | null; // クリックで確定した座標
    onCreated?: (p: ApiAreaPin) => void;            // 作成後に親へ返す
  };
};

export const AreaEditForm: React.FC<Props> = ({ onClose, context }) => {
  const [areaName, setAreaName] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const canSave = useMemo(() => {
    return !!areaName && !!context?.mapId && !!context?.draftPos && !submitting;
  }, [areaName, context?.mapId, context?.draftPos, submitting]);

  const handleSave = async () => {
    if (!canSave || !context) return;
    const { mapId, draftPos } = context;
    if (!mapId || !draftPos) return;

    setSubmitting(true);
    try {
      // 1) ★ 空マップ作成（swaggerでは MapCreateRequest は parentMapId のみ）
      //    parentMapId: null で root 作成（必要に応じて親を指定）
      const resMap = await fetch(`${API}/maps`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ parentMapId: context.mapId }),
      });
      if (!resMap.ok) {
        const t = await resMap.text();
        throw new Error(`POST /maps failed: ${resMap.status} ${t}`);
      }
      const createdMap = await resMap.json(); // { id: string, ... }
      const linkedMapId: string = createdMap?.id;

      // 2) ★ 親マップにエリアピン作成（エンドポイントは複数形 /pins）
      //    サーバ定義では category は必須。エリアは UI分類上 category 不要でも
      //    バリデーション通すため "plan" を付与（サーバ側は無視してもOK）。
      const body = {
        name: areaName,
        type: "area_selector",
        linkToMapId: linkedMapId,
        xNorm: draftPos.xNorm,
        yNorm: draftPos.yNorm,
        category: "plan",       // ★ 必須
        status: "open",
        waitMinutes: 0,
      };
      const resPin = await fetch(`${API}/maps/${mapId}/pins`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!resPin.ok) {
        const t = await resPin.text();
        throw new Error(`POST /maps/${mapId}/pins failed: ${resPin.status} ${t}`);
      }
      const createdPin = await resPin.json();

      // 親に返してリストへ追加（AreaPin 形へ整形）
      const areaPin: ApiAreaPin = {
        id: createdPin.id,
        mapId: createdPin.mapId,
        name: createdPin.name,
        xNorm: createdPin.xNorm,
        yNorm: createdPin.yNorm,
        linkToMapId: createdPin.linkToMapId ?? linkedMapId,
      };
      context.onCreated?.(areaPin);

      alert("エリアピンを作成しました。");
      onClose();
    } catch (e: any) {
      console.error(e);
      alert(`エリアピンの作成に失敗しました: ${e.message ?? e}`);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title">ピンを編集（エリア）</h2>

      <div className="mb-2 text-xs text-gray-500">
        親マップ: <code>{context?.mapId ?? "(未選択)"}</code>
      </div>
      <div className="mb-4 text-xs text-gray-500">
        座標:{" "}
        <code>
          x={(context?.draftPos?.xNorm ?? 0).toFixed(3)}, y={(context?.draftPos?.yNorm ?? 0).toFixed(3)}
        </code>
      </div>

      <AreaNameForm value={areaName} onChange={setAreaName} />

      <div className="buttonRow">
        <SaveButton disabled={!canSave} onClick={handleSave} />
        <DeleteButton onClick={onClose} />
      </div>
    </div>
  );
};
