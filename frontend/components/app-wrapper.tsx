'use client'

import { useState } from 'react'
import { useInitialization } from '@/hooks/use-initialization'
import { InitializationScreen } from '@/components/initialization-screen'

interface AppWrapperProps {
  children: React.ReactNode
}

export function AppWrapper({ children }: AppWrapperProps) {
  const [initializationComplete, setInitializationComplete] = useState(false)
  const { isInitializing, isFailed, isCompleted, isLoading } = useInitialization()

  // Show initialization screen if:
  // 1. Still loading initial status
  // 2. Currently initializing 
  // 3. Failed initialization
  // 4. Not yet marked as complete by user interaction
  if (!initializationComplete && (isLoading || isInitializing || isFailed || !isCompleted)) {
    return (
      <InitializationScreen 
        onComplete={() => setInitializationComplete(true)} 
      />
    )
  }

  // Show main app once initialization is complete and user has seen the completion screen
  return <>{children}</>
}