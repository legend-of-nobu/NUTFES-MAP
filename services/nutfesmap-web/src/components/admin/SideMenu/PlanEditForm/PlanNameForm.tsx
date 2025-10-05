"use client";
import React from "react";
import styles from "./PlanEditForm.module.css";

type PlanNameFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function PlanNameForm({ value, onChange }: PlanNameFormProps) {
  return (
    <div>
      <label className={styles.planNameLabel}>企画名</label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="企画名を入力"
        className={styles.planNameForm}
      />
    </div>
  );
}
