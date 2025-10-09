"use client";
import React from "react";
import "../FormStyle.css";
import "../Style.css";

type PlanPlaceFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function PlanPlaceForm({ value, onChange }: PlanPlaceFormProps) {
  return (
    <div>
      <label className="Label">場所（表示用）</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="例: 学生ホール"
        className="Form"
      />
    </div>
  );
}
