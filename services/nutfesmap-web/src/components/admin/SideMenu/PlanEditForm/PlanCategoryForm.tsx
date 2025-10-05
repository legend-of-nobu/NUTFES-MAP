"use client";
import React from "react";
import  "../FormStyle.css";
import  "../Style.css";

type PlanCategoryFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function PlanCategoryForm({ value, onChange }: PlanCategoryFormProps) {
  return (
    <div>
      <label className="Label">カテゴリ</label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="categorySelect">
        <option value="">選択してください</option>
        <option value="food">飲食</option>
        <option value="plan">企画</option>
        <option value="forChild">子供向け</option>
      </select>
    </div>
  );
}