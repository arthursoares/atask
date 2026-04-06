import type { ButtonHTMLAttributes, ReactNode } from "react";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "sm" | "default" | "lg";

export interface ButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
}

export function buttonClassName(variant: ButtonVariant, size: ButtonSize) {
  return [
    "ui-btn",
    `ui-btn-${variant}`,
    size !== "default" ? `ui-btn-${size}` : "",
  ].filter(Boolean).join(" ");
}

export default function Button({
  children,
  variant = "secondary",
  size = "default",
  className,
  type = "button",
  ...props
}: ButtonProps) {
  const classes = [buttonClassName(variant, size), className ?? ""].filter(Boolean).join(" ");

  return (
    <button type={type} className={classes} {...props}>
      {children}
    </button>
  );
}
