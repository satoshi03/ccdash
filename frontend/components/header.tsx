"use client"

import { useState } from "react"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { LanguageSelector } from "@/components/language-selector"
import { SettingsModal } from "@/components/settings-modal"
import { useI18n } from "@/hooks/use-i18n"
import { useSyncLogs } from "@/hooks/use-api"
import { Settings } from "@/lib/settings"
import { 
  Menu, 
  RefreshCw, 
  Github,
  Zap
} from "lucide-react"

interface HeaderProps {
  onSettingsChange?: (settings: Settings) => void
}

export function Header({ onSettingsChange }: HeaderProps) {
  const { t, language, changeLanguage } = useI18n()
  const { sync: syncLogs } = useSyncLogs()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  const handleRefresh = async () => {
    setIsRefreshing(true)
    try {
      await syncLogs()
      // ページをリロードしてデータを更新
      window.location.reload()
    } catch (error) {
      console.error('Error syncing data:', error)
    } finally {
      setIsRefreshing(false)
    }
  }


  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto max-w-7xl flex h-16 items-center px-6">
        {/* Logo */}
        <div className="mr-6 flex items-center space-x-2">
          <Link href="/" className="flex items-center space-x-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
              <Zap className="h-4 w-4 text-primary-foreground" />
            </div>
            <div className="hidden font-bold sm:inline-block">
              {t('header.title')}
            </div>
          </Link>
          <Badge variant="secondary" className="hidden sm:inline-flex">
            v{process.env.NEXT_PUBLIC_APP_VERSION || '0.1.0'}
          </Badge>
        </div>


        {/* Right side controls */}
        <div className="flex flex-1 items-center justify-end space-x-2">
          <div className="hidden items-center space-x-2 sm:flex">
            <Button variant="ghost" size="sm" asChild>
              <Link href="https://github.com/satoshi03/claudeee" target="_blank" rel="noopener noreferrer">
                <Github className="h-4 w-4" />
                <span className="sr-only">GitHub</span>
              </Link>
            </Button>
            <Button onClick={handleRefresh} disabled={isRefreshing} variant="outline" size="sm">
              <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
              {t('common.refresh')}
            </Button>
            <LanguageSelector currentLanguage={language} onLanguageChange={changeLanguage} />
            <SettingsModal onSettingsChange={onSettingsChange} />
          </div>

          {/* Mobile Menu */}
          <Sheet open={isMobileMenuOpen} onOpenChange={setIsMobileMenuOpen}>
            <SheetTrigger asChild className="md:hidden">
              <Button variant="ghost" size="sm" className="ml-2 px-0 text-base hover:bg-transparent focus-visible:bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0">
                <Menu className="h-6 w-6" />
                <span className="sr-only">Toggle Menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="right" className="pr-0">
              <SheetHeader>
                <SheetTitle className="flex items-center space-x-2">
                  <Zap className="h-5 w-5" />
                  <span>{t('header.title')}</span>
                </SheetTitle>
                <SheetDescription>
                  {t('header.subtitle')}
                </SheetDescription>
              </SheetHeader>
              <div className="my-4 h-px bg-border" />
              <div className="flex flex-col space-y-3">
                <Button variant="ghost" size="sm" asChild className="justify-start">
                  <Link href="https://github.com/claudeee/claudeee" target="_blank" rel="noopener noreferrer">
                    <Github className="h-4 w-4 mr-2" />
                    GitHub
                  </Link>
                </Button>
                <Button onClick={handleRefresh} disabled={isRefreshing} variant="outline" size="sm" className="justify-start">
                  <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
                  {t('common.refresh')}
                </Button>
                <LanguageSelector currentLanguage={language} onLanguageChange={changeLanguage} />
                <SettingsModal onSettingsChange={onSettingsChange} />
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  )
}

