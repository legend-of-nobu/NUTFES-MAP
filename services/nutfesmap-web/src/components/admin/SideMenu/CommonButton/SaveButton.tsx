"use client";
import React from "react";
import styles from "./Button.module.css";

export const SaveButton: React.FC<{ onClick?: () => void }> = ({ onClick }) => (
<button className={styles.saveButton} onClick={onClick}>
  保存
</button>
);
