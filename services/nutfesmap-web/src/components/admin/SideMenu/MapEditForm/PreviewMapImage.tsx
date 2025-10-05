"use client";
import React from "react";
import styles from "./MapEditForm.module.css"; 

type PreviewMapImageProps = {
  image: string;
};

export default function PreviewMapImage({ image }: PreviewMapImageProps) {
  return (
    <div className={styles.previewContainer}>
      <img src={image} alt="マップ画像プレビュー" className={styles.preview} />
    </div>
  );
}
