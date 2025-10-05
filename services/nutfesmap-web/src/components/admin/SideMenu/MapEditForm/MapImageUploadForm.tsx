import React from "react";
import styles from "./MapEditForm.module.css";

export const MapImageUploadForm: React.FC<{
  value: File | null;
  onChange: (file: File | null) => void;
}> = ({ value, onChange }) => (
  <div className={styles.mapImageUploadFormContainer}>
    
    <input
      type="file"
      accept="image/*"
      onChange={(e) => onChange(e.target.files?.[0] || null)}
      className={styles.mapImageInput}
    />
  </div>
);
