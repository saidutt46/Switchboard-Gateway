export function DetailSkeleton() {
  return (
    <div className="animate-pulse space-y-6">
      {/* Status bar */}
      <div className="flex items-center gap-3">
        <div className="h-5 w-16 rounded-full bg-muted" />
        <div className="h-4 w-48 rounded bg-muted" />
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="rounded-xl border bg-card p-4">
            <div className="flex items-center justify-between">
              <div className="space-y-2">
                <div className="h-3 w-14 rounded bg-muted" />
                <div className="h-7 w-10 rounded bg-muted" />
              </div>
              <div className="h-9 w-9 rounded-lg bg-muted" />
            </div>
          </div>
        ))}
      </div>

      {/* Two column layout */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-6">
          {/* Main card */}
          <div className="rounded-xl border bg-card overflow-hidden">
            <div className="border-b px-5 py-3.5">
              <div className="h-3 w-20 rounded bg-muted" />
            </div>
            <div className="p-5 space-y-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="flex items-center justify-between">
                  <div className="h-4 rounded bg-muted" style={{ width: `${40 + i * 15}%` }} />
                  <div className="h-4 w-16 rounded bg-muted" />
                </div>
              ))}
            </div>
          </div>
          {/* Second card */}
          <div className="rounded-xl border bg-card overflow-hidden">
            <div className="border-b px-5 py-3.5">
              <div className="h-3 w-16 rounded bg-muted" />
            </div>
            <div className="p-5 space-y-3">
              {[1, 2].map((i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className="h-4 w-24 rounded bg-muted" />
                  <div className="h-5 w-14 rounded-full bg-muted" />
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <div className="rounded-xl border bg-card overflow-hidden">
            <div className="border-b px-5 py-3.5">
              <div className="h-3 w-24 rounded bg-muted" />
            </div>
            <div className="divide-y">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div key={i} className="flex items-center justify-between px-5 py-2.5">
                  <div className="h-3.5 w-20 rounded bg-muted" />
                  <div className="h-3.5 w-16 rounded bg-muted" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
