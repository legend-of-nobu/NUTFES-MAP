"use client";
import React, { ChangeEvent } from "react";
import styles from "./PlanImageForm.module.css"; 

type UploadPlanImageProps = {
  onUpload: (imageUrl: string) => void;
};

export default function UploadPlanImage({ onUpload }: UploadPlanImageProps) {
  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onloadend = () => {
      if (reader.result) onUpload(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  return (
    <input
      type="file"
      accept="image/*"
      onChange={handleFileChange}
      className={styles.fileInput}
    />
  );
}
