"use client"

import { Suspense } from "react"
import Dashboard from "./dashboard-content"

function LoadingFallback() {
  return (
    <div className="container mx-auto max-w-7xl p-6 space-y-6">
      <div className="animate-pulse">
        <div className="bg-gray-200 rounded-lg h-32 mb-6"></div>
        <div className="bg-gray-200 rounded-lg h-64"></div>
      </div>
    </div>
  )
}

export default function DashboardWrapper() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <Dashboard />
    </Suspense>
  )
}