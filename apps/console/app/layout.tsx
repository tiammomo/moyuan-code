import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Moyuan 控制台",
  description: "多 Agent 代码生命周期控制台",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
