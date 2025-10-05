"use client";
import React from "react";
import styles from "./PlanEditForm.module.css";

type PlanExplainFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function PlanExplainForm({ value, onChange }: PlanExplainFormProps) {
  return (
    <div>
      <label className={styles.planExplainLabel}>企画の説明</label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="企画の説明を入力"
        rows={3}
        className={styles.planExplainForm}
      />
    </div>
  );
}
 