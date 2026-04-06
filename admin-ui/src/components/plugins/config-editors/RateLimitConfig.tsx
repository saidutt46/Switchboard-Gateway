import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function RateLimitConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Algorithm</label>
        <select {...register('config.algorithm')} defaultValue="token-bucket" className={inputClass}>
          <option value="token-bucket">Token Bucket</option>
          <option value="sliding-window">Sliding Window</option>
        </select>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>Limit</label>
          <input type="number" {...register('config.limit', { valueAsNumber: true })} defaultValue={100} placeholder="100" className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Window</label>
          <select {...register('config.window')} defaultValue="1m" className={inputClass}>
            <option value="1s">1 second</option>
            <option value="10s">10 seconds</option>
            <option value="1m">1 minute</option>
            <option value="10m">10 minutes</option>
            <option value="1h">1 hour</option>
            <option value="24h">24 hours</option>
          </select>
        </div>
      </div>
      <div>
        <label className={labelClass}>Identifier</label>
        <select {...register('config.identifier')} defaultValue="auto" className={inputClass}>
          <option value="auto">{'Auto (consumer > api_key > ip)'}</option>
          <option value="consumer_id">Consumer ID</option>
          <option value="api_key">API Key</option>
          <option value="ip">IP Address</option>
          <option value="shared">Shared (global counter)</option>
        </select>
      </div>
      <div>
        <label className={labelClass}>Response Code</label>
        <input type="number" {...register('config.response_code', { valueAsNumber: true })} defaultValue={429} className={inputClass} />
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.headers')} defaultChecked id="rate-headers" className="rounded border-input" />
        <label htmlFor="rate-headers" className="text-sm text-foreground">Include rate limit headers in response</label>
      </div>
      <div>
        <label className={labelClass}>Redis URL</label>
        <input {...register('config.redis_url')} defaultValue="redis://redis:6379/0" placeholder="redis://redis:6379/0" className={inputClass} />
        <p className="text-xs text-muted-foreground mt-1">Use <code className="font-mono">redis://redis:6379/0</code> for Docker, <code className="font-mono">redis://localhost:6379/0</code> for local</p>
      </div>
    </div>
  )
}
