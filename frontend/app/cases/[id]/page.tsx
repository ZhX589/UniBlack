'use client'

import { useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { LoadingState } from '@/components/ui/loading-state'

export default function LegacyCaseDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params?.id as string

  useEffect(() => {
    if (!id) return
    router.replace(`/events/${id}`)
  }, [id, router])

  return <LoadingState message="案件接口已弃用，正在跳转到事件详情..." />
}
