import React from "react";
import styles from "./MapEditForm.module.css";
import { Upload } from "lucide-react"; 

export const MapImageUploadForm: React.FC<{
  value: File | null;
  onChange: (file: File | null) => void;
}> = ({ value, onChange }) => (
  <div className={styles.mapImageUploadFormContainer}>
     <label htmlFor="mapUpload" className={styles.mapUploadLabel}>
        <Upload size={20} />
        <span>マップをアップロード</span>
      </label>
    <input
    id="mapUpload"
      type="file"
      accept="image/*"
      onChange={(e) => onChange(e.target.files?.[0] || null)}
      className={styles.mapImageInput}
    />
  </div>
);
