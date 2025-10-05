"use client";
import React from "react";
import styles from "./PlanImageForm.module.css"; 

type PreviewPlanImageProps = {
  image: string;
};

export default function PreviewPlanImage({ image }: PreviewPlanImageProps) {
  return (
    <div className={styles.previewContainer}>
      <img src={image} alt="企画画像プレビュー" className={styles.preview} />
    </div>
  );
}
