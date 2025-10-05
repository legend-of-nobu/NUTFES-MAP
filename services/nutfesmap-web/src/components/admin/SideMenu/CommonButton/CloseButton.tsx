"use client";
import React from "react";
import styles from "./Button.module.css";

export const CloseButton: React.FC<{ onClick: () => void }> = ({ onClick }) => (
  <button className={styles.closeButton} onClick={onClick}>
     ✕ 
  </button>
);




