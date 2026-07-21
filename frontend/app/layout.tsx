import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from './providers'
import { SiteHeader } from '@/components/shell/site-header'

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
        <Providers><SiteHeader /><main className="container mx-auto p-4">{children}</main><footer className="bg-gray-900 text-white p-4 mt-8"><div className="container mx-auto text-center"><p>UniBlack - 云黑名单系统</p></div></footer></Providers>
      </body>
    </html>
  )
}
