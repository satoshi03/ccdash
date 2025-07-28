'use client'

import { useInitialization } from '@/hooks/use-initialization'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import { AlertCircle, CheckCircle, Database, FileText, Loader2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface InitializationScreenProps {
  onComplete?: () => void
}

export function InitializationScreen({ onComplete }: InitializationScreenProps) {
  const { status, isLoading, error, retry, isInitializing, isCompleted, isFailed } = useInitialization()

  // Auto-complete when initialization is done
  if (isCompleted && onComplete) {
    // If already completed, show success screen briefly then auto-transition
    setTimeout(() => onComplete(), 1500)
  }

  if (isLoading && !status) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-blue-100">
              <Loader2 className="h-6 w-6 animate-spin text-blue-600" />
            </div>
            <CardTitle>Connecting...</CardTitle>
            <CardDescription>Checking system status</CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  if (error && !status) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-red-50 to-pink-100">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
              <AlertCircle className="h-6 w-6 text-red-600" />
            </div>
            <CardTitle>Connection Error</CardTitle>
            <CardDescription>Failed to connect to the server</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
            <Button className="w-full" onClick={retry}>
              Retry Connection
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isInitializing) {
    const progress = status?.progress
    const progressPercentage = progress ? 
      (progress.total_files > 0 ? (progress.processed_files / progress.total_files) * 100 : 0) : 0

    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-lg">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-blue-100">
              <Database className="h-8 w-8 text-blue-600" />
            </div>
            <CardTitle className="text-2xl">Initializing Database</CardTitle>
            <CardDescription>{status?.message}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {progress && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Processing log files...</span>
                    <span>{progress.processed_files} / {progress.total_files}</span>
                  </div>
                  <Progress value={progressPercentage} className="h-2" />
                </div>
                
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center space-x-2">
                    <FileText className="h-4 w-4 text-blue-500" />
                    <span>Files: {progress.processed_files}</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Database className="h-4 w-4 text-green-500" />
                    <span>Lines: {progress.new_lines}</span>
                  </div>
                </div>
              </div>
            )}
            
            <div className="flex items-center justify-center space-x-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span>This may take a few moments for large log files...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isFailed) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-red-50 to-pink-100">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
              <AlertCircle className="h-6 w-6 text-red-600" />
            </div>
            <CardTitle>Initialization Failed</CardTitle>
            <CardDescription>{status?.message}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {status?.error && (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{status.error}</AlertDescription>
              </Alert>
            )}
            <Button className="w-full" onClick={retry}>
              Retry Initialization
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isCompleted) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-green-50 to-emerald-100">
        <Card className="w-full max-w-md">
          <CardHeader className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
              <CheckCircle className="h-6 w-6 text-green-600" />
            </div>
            <CardTitle>Initialization Complete</CardTitle>
            <CardDescription>{status?.message}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {status?.progress && (
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="text-center p-3 bg-white rounded-lg">
                  <div className="font-semibold text-lg">{status.progress.processed_files}</div>
                  <div className="text-muted-foreground">Files Processed</div>
                </div>
                <div className="text-center p-3 bg-white rounded-lg">
                  <div className="font-semibold text-lg">{status.progress.new_lines}</div>
                  <div className="text-muted-foreground">Log Entries</div>
                </div>
              </div>
            )}
            <div className="flex items-center justify-center space-x-2 text-sm text-muted-foreground">
              <CheckCircle className="h-4 w-4 text-green-500" />
              <span>Ready to use! Redirecting...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return null
}