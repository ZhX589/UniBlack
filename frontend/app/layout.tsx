import type { Metadata } from 'next'
import './globals.css'
import { Providers } from './providers'
import { SiteHeader } from '@/components/shell/site-header'
import { SiteFooter } from '@/components/shell/site-footer'

export const metadata: Metadata = {
  title: 'UniBlack',
  description: '一个可复用的通用云黑系统',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className="font-sans antialiased">
        <Providers>
          <SiteHeader />
          <main className="container mx-auto min-h-[70vh] p-4">{children}</main>
          <SiteFooter />
        </Providers>
      </body>
    </html>
  )
}
