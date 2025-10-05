"use client";
import React from "react";
import styles from "./Button.module.css";

export const DeleteButton: React.FC<{ onClick?: () => void }> = ({ onClick }) => (
<button className={styles.deleteButton} onClick={onClick}>
  削除
</button>
);