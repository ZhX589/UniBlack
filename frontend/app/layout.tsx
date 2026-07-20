import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'UniBlack - 云黑名单系统',
  description: '一个可复用的通用云黑系统',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="zh-CN">
      <body className={inter.className}>
        <nav className="bg-gray-900 text-white p-4">
          <div className="container mx-auto flex justify-between items-center">
            <a href="/" className="text-xl font-bold">UniBlack</a>
            <div className="flex gap-4">
              <a href="/search" className="hover:text-gray-300">查询</a>
              <a href="/subjects" className="hover:text-gray-300">黑名单</a>
              <a href="/submit" className="hover:text-gray-300">举报</a>
              <a href="/admin" className="hover:text-gray-300">管理</a>
              <a href="/login" className="hover:text-gray-300">登录</a>
            </div>
          </div>
        </nav>
        <main className="container mx-auto p-4">
          {children}
        </main>
        <footer className="bg-gray-900 text-white p-4 mt-8">
          <div className="container mx-auto text-center">
            <p>UniBlack - 云黑名单系统</p>
          </div>
        </footer>
      </body>
    </html>
  )
}
