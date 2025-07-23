"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Languages, Check } from "lucide-react"
import { Language, loadLanguage } from "@/lib/i18n"

interface LanguageSelectorProps {
  currentLanguage?: Language
  onLanguageChange?: (language: Language) => void
}

export function LanguageSelector({ currentLanguage: propCurrentLanguage, onLanguageChange }: LanguageSelectorProps) {
  const [currentLanguage, setCurrentLanguage] = useState<Language>('ja')

  useEffect(() => {
    if (propCurrentLanguage) {
      setCurrentLanguage(propCurrentLanguage)
    } else {
      const savedLanguage = loadLanguage()
      setCurrentLanguage(savedLanguage)
    }
  }, [propCurrentLanguage])

  const handleLanguageChange = (language: Language) => {
    setCurrentLanguage(language)
    // 親コンポーネントに変更を通知（useI18nのchangeLanguageが呼ばれる）
    onLanguageChange?.(language)
  }

  const languages = [
    { code: 'ja' as Language, name: '日本語', flag: '🇯🇵' },
    { code: 'en' as Language, name: 'English', flag: '🇺🇸' },
  ]

  const currentLang = languages.find(lang => lang.code === currentLanguage)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2">
          <Languages className="w-4 h-4" />
          <span>{currentLang?.flag}</span>
          <span className="hidden sm:inline">{currentLang?.name}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {languages.map((language) => (
          <DropdownMenuItem
            key={language.code}
            onClick={() => handleLanguageChange(language.code)}
            className="flex items-center justify-between gap-2"
          >
            <div className="flex items-center gap-2">
              <span>{language.flag}</span>
              <span>{language.name}</span>
            </div>
            {currentLanguage === language.code && (
              <Check className="w-4 h-4" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}