"use client";
import React, { useState } from "react";
import UploadPlanImage from "./UploadPlanImage";
import PreviewPlanImage from "./PreviewPlanImage";
import styles from "./PlanImageForm.module.css";

interface PlanImageFormProps {
  value: string | null;
  onChange: (image: string | null) => void;
}

export default function PlanImageForm({ value, onChange }: PlanImageFormProps) {
  const [image, setImage] = useState<string | null>(value);

  const handleUpload = (uploadedImage: string | null) => {
    setImage(uploadedImage);
    onChange(uploadedImage);
  };

  return (
    <div className={styles.container}>
      <label className={styles.uploadImageLabel}>マップをアップロード</label>
      <div className={styles.fileinputContainer}>
         <UploadPlanImage onUpload={handleUpload} />
         {image && (
            <div className={styles.previewWrapper}>
          <PreviewPlanImage image={image} />
            </div>
        
             )}
        </div>
    </div>
  );
}

