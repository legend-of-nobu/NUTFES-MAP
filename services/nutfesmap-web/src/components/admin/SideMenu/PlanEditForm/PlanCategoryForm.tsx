"use client";
import React from "react";
import  "../FormStyle.css";
import  "../Style.css";
import  { Category } from "@/types/enums";

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
        onChange={(e) => onChange(e.target.value as Category)}
        className="categorySelect">
        <option value="">選択してください</option>
        <option value="{Category.Food}">{Category.Food}</option>
        <option value="{Category.Plan}">{Category.Plan}</option>
        <option value="{Category.Child}">{Category.Child}</option>
      </select>
    </div>
  );
}