"use client";
import React from "react";
import "../FormStyle.css";
import "../Style.css";

type PlanNameFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function PlanNameForm({ value, onChange }: PlanNameFormProps) {
  return (
    <div>
      <label className="Label">企画名</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="企画名を入力"
        className="Form"
      />
    </div>
  );
}
