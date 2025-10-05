"use client";
import React from "react";
import styles from "./PlanEditForm.module.css";

type PlanClosedFormProps = {
  value: boolean;
  onChange: (val: boolean) => void;
};

export default function PlanClosedForm({ value, onChange }: PlanClosedFormProps) {
  return (
    <div className={styles.planClosedForm}>
        <label className={styles.checkboxLabel}>営業状況</label>
    <div className={styles.checkboxContainer}>
        
      <input
        type="checkbox"
        checked={value}
        onChange={(e) => onChange(e.target.checked)}
        className={styles.checkbox}
      />
      <label className={styles.checkboxName}>営業終了</label>
    </div>
    </div>
  );
}
