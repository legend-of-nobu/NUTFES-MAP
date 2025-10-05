"use client";
import React from "react";
import styles from "./PlanEditForm.module.css";

type WaitTimeFormProps = {
  value: string;
  onChange: (val: string) => void;
};

export default function WaitTimeForm({ value, onChange }: WaitTimeFormProps) {
  return (
    <div>
      <label className={styles.waitTimeLabel}>待ち時間</label>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="待ち時間を入力"
        className={styles.waitTimeForm}
      />
    </div>
  );
}
