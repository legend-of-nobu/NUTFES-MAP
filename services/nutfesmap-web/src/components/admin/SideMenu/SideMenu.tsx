"use client";
import React from "react";
import { PlanEditForm } from "./PlanEditForm/PlanEditForm";
import { AreaEditForm } from "./AreaEditForm/AreaEditForm";
import { MapEditForm, MapEditFormProps } from "./MapEditForm/MapEditForm";

type SideMenuProps = {
  mode: "plan" | "area" | "map";
  onClose: () => void;
  mapEditProps?: Omit<MapEditFormProps, "onClose">; // onClose は SideMenu から渡す
};

export default function SideMenu({ mode, onClose, mapEditProps }: SideMenuProps) {
  return (
    <div>
      {mode === "plan" && <PlanEditForm onClose={onClose} />}
      {mode === "area" && <AreaEditForm onClose={onClose} />}
      {mode === "map" && (
        <MapEditForm
          {...(mapEditProps ?? ({} as any))}
          onClose={onClose}
        />
      )}
    </div>
  );
}
