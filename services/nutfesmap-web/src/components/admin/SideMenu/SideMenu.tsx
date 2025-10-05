"use client";
import React from "react";
import { PlanEditForm } from "./PlanEditForm/PlanEditForm";
import { AreaEditForm } from "./AreaEditForm/AreaEditForm";
import { MapEditForm } from "./MapEditForm/MapEditForm";

type SideMenuProps = {
  mode: "plan" | "area" | "map";
  onClose: () => void;
};

export default function SideMenu({ mode, onClose }: SideMenuProps) {
  return (
    <div>
      {/* 内容切り替え */}
      {mode === "plan" && <PlanEditForm onClose={onClose}/>}
        {mode === "area" && <AreaEditForm onClose={onClose} />}
      {mode === "map" && <MapEditForm onClose={onClose} />}
    </div>
  );
}
