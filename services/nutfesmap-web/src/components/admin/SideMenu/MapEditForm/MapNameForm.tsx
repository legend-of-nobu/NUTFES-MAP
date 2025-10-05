import React from "react";
import styles from "./MapEditForm.module.css";

export const MapNameForm: React.FC<{
  value: string;
  onChange: (value: string) => void;
}> = ({ value, onChange }) => (
  <div className={styles.mapNameFormContainer}>
    <label className={styles.mapNameLabel}>マップ名</label>
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="マップ名を入力"
      className={styles.mapNameForm}
    />
  </div>
);