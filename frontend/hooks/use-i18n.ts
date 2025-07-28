"use client"

import { useState, useEffect, useCallback } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { Language, getTranslation, loadLanguage, formatDateTime, formatFullDateTime } from "@/lib/i18n"

export function useI18n() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [language, setLanguage] = useState<Language>('ja')

  useEffect(() => {
    // URLパラメータから言語を取得
    const localeParam = searchParams.get('locale') as Language
    
    if (localeParam === 'en' || localeParam === 'ja') {
      setLanguage(localeParam)
    } else {
      // LocalStorageまたはデフォルト値を使用
      const savedLanguage = loadLanguage()
      setLanguage(savedLanguage)
    }
  }, [searchParams])

  const t = useCallback((key: string): string => {
    return getTranslation(language, key)
  }, [language])

  const changeLanguage = useCallback((newLanguage: Language) => {
    setLanguage(newLanguage)
    
    // LocalStorageに保存
    if (typeof window !== 'undefined') {
      localStorage.setItem('ccdash-language', newLanguage)
    }
    
    // URLパラメータを更新
    const currentUrl = new URL(window.location.href)
    if (newLanguage === 'ja') {
      currentUrl.searchParams.delete('locale')
    } else {
      currentUrl.searchParams.set('locale', newLanguage)
    }
    
    router.push(currentUrl.pathname + currentUrl.search)
  }, [router])

  const formatDate = useCallback((date: Date): string => {
    return formatDateTime(language, date)
  }, [language])

  const formatFullDate = useCallback((date: Date): string => {
    return formatFullDateTime(language, date)
  }, [language])

  return {
    language,
    t,
    changeLanguage,
    formatDate,
    formatFullDate,
  }
}