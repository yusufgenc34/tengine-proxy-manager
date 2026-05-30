import { useState } from 'react'
import { Download, Copy, Check, Monitor } from 'lucide-react'
import toast from 'react-hot-toast'

interface Props {
  domain: string
  certId: number
  onClose: () => void
}

export default function DownloadGuideDialog({ domain, certId, onClose }: Props) {
  const [step, setStep] = useState<'guide' | 'content'>('guide')
  const [certContent, setCertContent] = useState('')
  const [keyContent, setKeyContent] = useState('')
  const [copiedCert, setCopiedCert] = useState(false)
  const [copiedKey, setCopiedKey] = useState(false)
  const [loading, setLoading] = useState(false)

  const fetchAndDownload = async () => {
    setLoading(true)
    try {
      const { default: api } = await import('../api/client')
      const { data } = await api.get(`/certificates/${certId}/download`)
      setCertContent(data.cert_content)
      setKeyContent(data.key_content)
      setStep('content')
    } catch {
      toast.error('Failed to fetch certificate')
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = async (text: string, setCopied: (v: boolean) => void) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('Failed to copy')
    }
  }

  const downloadFile = (content: string, filename: string) => {
    const blob = new Blob([content], { type: 'application/x-pem-file' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 max-h-[80vh] overflow-y-auto">
        <div className="flex items-center gap-3 mb-4">
          <div className="p-2 bg-purple-100 rounded-full">
            <Monitor size={20} className="text-purple-600" />
          </div>
          <div>
            <h3 className="font-semibold">Install Certificate: {domain}</h3>
            <p className="text-sm text-gray-500">Trust this self-signed certificate on your system</p>
          </div>
        </div>

        {step === 'guide' ? (
          <>
            {/* macOS */}
            <div className="mb-4 p-3 bg-gray-50 rounded-lg">
              <p className="text-sm font-medium mb-2">macOS</p>
              <p className="text-xs text-gray-500 mb-2">
                Download the certificate file, then run:
              </p>
              <code className="block text-xs bg-gray-900 text-green-400 p-2 rounded overflow-x-auto">
                sudo security add-trusted-cert -d -r trustRoot -p ssl -k /Library/Keychains/System.keychain ~/Downloads/{domain}-fullchain.pem
              </code>
            </div>

            {/* Windows */}
            <div className="mb-4 p-3 bg-gray-50 rounded-lg">
              <p className="text-sm font-medium mb-2">Windows</p>
              <p className="text-xs text-gray-500 mb-2">
                Run PowerShell as Administrator, then:
              </p>
              <code className="block text-xs bg-gray-900 text-green-400 p-2 rounded overflow-x-auto">
                Import-Certificate -FilePath "$env:USERPROFILE\Downloads\{domain}-fullchain.pem" -CertStoreLocation Cert:\LocalMachine\Root
              </code>
            </div>

            {/* Linux */}
            <div className="mb-4 p-3 bg-gray-50 rounded-lg">
              <p className="text-sm font-medium mb-2">Linux (Ubuntu/Debian)</p>
              <code className="block text-xs bg-gray-900 text-green-400 p-2 rounded overflow-x-auto">
                sudo cp {domain}-fullchain.pem /usr/local/share/ca-certificates/{domain}.crt{"\n"}
                sudo update-ca-certificates
              </code>
            </div>

            <div className="flex justify-end gap-2 border-t pt-4">
              <button onClick={onClose} className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50">
                Close
              </button>
              <button
                onClick={fetchAndDownload}
                disabled={loading}
                className="px-4 py-2 text-sm bg-purple-600 text-white rounded-lg hover:bg-purple-700 flex items-center gap-2 disabled:opacity-50"
              >
                <Download size={16} />
                {loading ? 'Loading...' : 'Download Files'}
              </button>
            </div>
          </>
        ) : (
          <>
            <div className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-sm font-medium">Certificate (fullchain.pem)</label>
                  <div className="flex gap-1">
                    <button
                      onClick={() => copyToClipboard(certContent, setCopiedCert)}
                      className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                    >
                      {copiedCert ? <Check size={12} className="text-green-600" /> : <Copy size={12} />}
                      {copiedCert ? 'Copied' : 'Copy'}
                    </button>
                    <button
                      onClick={() => downloadFile(certContent, `${domain}-fullchain.pem`)}
                      className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                    >
                      <Download size={12} /> Save
                    </button>
                  </div>
                </div>
                <textarea
                  readOnly
                  value={certContent}
                  rows={6}
                  className="w-full px-3 py-2 border rounded-lg font-mono text-xs bg-gray-50 resize-none"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-sm font-medium">Private Key (privkey.pem)</label>
                  <div className="flex gap-1">
                    <button
                      onClick={() => copyToClipboard(keyContent, setCopiedKey)}
                      className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                    >
                      {copiedKey ? <Check size={12} className="text-green-600" /> : <Copy size={12} />}
                      {copiedKey ? 'Copied' : 'Copy'}
                    </button>
                    <button
                      onClick={() => downloadFile(keyContent, `${domain}-privkey.pem`)}
                      className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                    >
                      <Download size={12} /> Save
                    </button>
                  </div>
                </div>
                <textarea
                  readOnly
                  value={keyContent}
                  rows={6}
                  className="w-full px-3 py-2 border rounded-lg font-mono text-xs bg-gray-50 resize-none"
                />
              </div>
            </div>

            <div className="flex justify-between gap-2 border-t pt-4 mt-4">
              <button onClick={() => setStep('guide')} className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50">
                Back
              </button>
              <button onClick={onClose} className="px-4 py-2 text-sm bg-purple-600 text-white rounded-lg hover:bg-purple-700">
                Done
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
