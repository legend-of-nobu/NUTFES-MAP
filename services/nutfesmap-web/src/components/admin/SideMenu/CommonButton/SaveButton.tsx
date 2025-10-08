"use client";
import React from "react";
import styles from "./Button.module.css";

type Props = {
  onClick?: () => void;
  disabled?: boolean;
};

export const SaveButton: React.FC<Props> = ({ onClick, disabled }) => (
  <button type="button" className={styles.saveButton} onClick={onClick} disabled={disabled}>
    保存
  </button>
);
