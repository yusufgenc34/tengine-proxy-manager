import { useState, useRef, type FormEvent } from 'react'
import { Upload, FileCheck, Copy, Check } from 'lucide-react'
import toast from 'react-hot-toast'
import api from '../api/client'
import Offcanvas from './Offcanvas'

interface Props {
  onClose: () => void
}

interface CreatedCert {
  domain: string
  cert_content: string
  key_content: string
}

export default function CertModal({ onClose }: Props) {
  const [tab, setTab] = useState<'letsencrypt' | 'custom' | 'self-signed'>('letsencrypt')
  const [customMode, setCustomMode] = useState<'upload' | 'text'>('upload')
  const [domain, setDomain] = useState('')
  const [certFile, setCertFile] = useState<File | null>(null)
  const [keyFile, setKeyFile] = useState<File | null>(null)
  const [certText, setCertText] = useState('')
  const [keyText, setKeyText] = useState('')
  const [saving, setSaving] = useState(false)
  const [createdCert, setCreatedCert] = useState<CreatedCert | null>(null)
  const [copiedCert, setCopiedCert] = useState(false)
  const [copiedKey, setCopiedKey] = useState(false)
  const certInputRef = useRef<HTMLInputElement>(null)
  const keyInputRef = useRef<HTMLInputElement>(null)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!domain.trim()) {
      toast.error('Domain is required')
      return
    }
    setSaving(true)
    try {
      if (tab === 'letsencrypt') {
        await api.post('/certificates/letsencrypt', { domain })
        toast.success('Certificate created')
        onClose()
      } else if (tab === 'self-signed') {
        const { data } = await api.post('/certificates/self-signed', { domain })
        setCreatedCert({
          domain: data.domain,
          cert_content: data.cert_content,
          key_content: data.key_content,
        })
        toast.success('Self-signed certificate created')
      } else if (customMode === 'upload') {
        if (!certFile || !keyFile) {
          toast.error('Both certificate and key files are required')
          setSaving(false)
          return
        }
        const formData = new FormData()
        formData.append('domain', domain)
        formData.append('cert_file', certFile)
        formData.append('key_file', keyFile)
        await api.post('/certificates/custom', formData, {
          headers: { 'Content-Type': 'multipart/form-data' },
        })
        toast.success('Certificate created')
        onClose()
      } else {
        if (!certText.trim() || !keyText.trim()) {
          toast.error('Both certificate and key content are required')
          setSaving(false)
          return
        }
        await api.post('/certificates/custom', {
          domain,
          cert_content: certText,
          key_content: keyText,
        })
        toast.success('Certificate created')
        onClose()
      }
    } catch (err: any) {
      const data = err.response?.data
      if (data?.code === 'LOCAL_DOMAIN') {
        toast.error(data.message, { duration: 8000 })
        setTab('self-signed')
      } else {
        toast.error(data?.message || 'Operation failed')
      }
    } finally {
      setSaving(false)
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

  const downloadAsFile = (content: string, filename: string) => {
    const blob = new Blob([content], { type: 'application/x-pem-file' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <Offcanvas title={createdCert ? 'Certificate Created' : 'New Certificate'} onClose={onClose}>
      {createdCert ? (
        <div className="space-y-4 mt-4">
          <div className="bg-green-50 border border-green-200 rounded-lg p-4">
            <p className="text-sm font-medium text-green-800">Self-signed certificate created successfully</p>
            <p className="text-xs text-green-600 mt-1">Domain: {createdCert.domain}</p>
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-sm font-medium">Certificate (fullchain.pem)</label>
              <div className="flex gap-1">
                <button
                  type="button"
                  onClick={() => copyToClipboard(createdCert.cert_content, setCopiedCert)}
                  className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                >
                  {copiedCert ? <Check size={12} className="text-green-600" /> : <Copy size={12} />}
                  {copiedCert ? 'Copied' : 'Copy'}
                </button>
                <button
                  type="button"
                  onClick={() => downloadAsFile(createdCert.cert_content, `${createdCert.domain}-fullchain.pem`)}
                  className="px-2 py-1 text-xs border rounded hover:bg-gray-50"
                >
                  Download
                </button>
              </div>
            </div>
            <textarea
              readOnly
              value={createdCert.cert_content}
              rows={6}
              className="w-full px-3 py-2 border rounded-lg font-mono text-xs bg-gray-50 resize-none"
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="text-sm font-medium">Private Key (privkey.pem)</label>
              <div className="flex gap-1">
                <button
                  type="button"
                  onClick={() => copyToClipboard(createdCert.key_content, setCopiedKey)}
                  className="flex items-center gap-1 px-2 py-1 text-xs border rounded hover:bg-gray-50"
                >
                  {copiedKey ? <Check size={12} className="text-green-600" /> : <Copy size={12} />}
                  {copiedKey ? 'Copied' : 'Copy'}
                </button>
                <button
                  type="button"
                  onClick={() => downloadAsFile(createdCert.key_content, `${createdCert.domain}-privkey.pem`)}
                  className="px-2 py-1 text-xs border rounded hover:bg-gray-50"
                >
                  Download
                </button>
              </div>
            </div>
            <textarea
              readOnly
              value={createdCert.key_content}
              rows={6}
              className="w-full px-3 py-2 border rounded-lg font-mono text-xs bg-gray-50 resize-none"
            />
          </div>

          <div className="flex justify-end border-t pt-4">
            <button
              onClick={() => { onClose() }}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Done
            </button>
          </div>
        </div>
      ) : (
        <>
          <div className="flex border-b -mx-6 px-6">
            <button
              onClick={() => setTab('letsencrypt')}
              className={`flex-1 py-2 text-sm font-medium ${tab === 'letsencrypt' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500'}`}
            >
              Let's Encrypt
            </button>
            <button
              onClick={() => setTab('custom')}
              className={`flex-1 py-2 text-sm font-medium ${tab === 'custom' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500'}`}
            >
              Custom Upload
            </button>
            <button
              onClick={() => setTab('self-signed')}
              className={`flex-1 py-2 text-sm font-medium ${tab === 'self-signed' ? 'border-b-2 border-purple-600 text-purple-600' : 'text-gray-500'}`}
            >
              Self-Signed
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4 mt-4">
            <div>
              <label className="block text-sm font-medium mb-1">Domain</label>
              <input
                type="text"
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
                placeholder="e.g. example.com"
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
              {tab === 'letsencrypt' && (
                <p className="text-xs text-gray-400 mt-1">Domain must point to this server for validation to work.</p>
              )}
              {tab === 'self-signed' && (
                <div className="mt-3 space-y-3">
                  <div className="bg-purple-50 border border-purple-200 rounded-lg p-3">
                    <p className="text-xs text-purple-700">
                      Create a self-signed certificate for local domains (.local, .test, .localhost) and IP addresses.
                      These certificates are not trusted by browsers and should only be used in development environments.
                    </p>
                  </div>
                  <div className="bg-yellow-50 border border-yellow-200 rounded p-3">
                    <p className="text-xs text-yellow-700">
                      Warning: Self-signed certificates will cause browser security warnings.
                    </p>
                  </div>
                </div>
              )}
            </div>

            {tab === 'custom' && (
              <>
                <div className="flex rounded-lg bg-gray-100 p-0.5">
                  <button
                    type="button"
                    onClick={() => setCustomMode('upload')}
                    className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-colors ${
                      customMode === 'upload' ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'
                    }`}
                  >
                    File Upload
                  </button>
                  <button
                    type="button"
                    onClick={() => setCustomMode('text')}
                    className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-colors ${
                      customMode === 'text' ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'
                    }`}
                  >
                    Paste Text
                  </button>
                </div>

                {customMode === 'upload' ? (
                  <>
                    <div>
                      <label className="block text-sm font-medium mb-1">Certificate File (.pem, .crt)</label>
                      <input
                        ref={certInputRef}
                        type="file"
                        accept=".pem,.crt,.cer"
                        onChange={(e) => setCertFile(e.target.files?.[0] ?? null)}
                        className="hidden"
                      />
                      <button
                        type="button"
                        onClick={() => certInputRef.current?.click()}
                        className={`w-full flex items-center gap-3 px-4 py-3 border-2 border-dashed rounded-lg transition-colors ${
                          certFile ? 'border-green-300 bg-green-50' : 'border-gray-300 hover:border-blue-400 hover:bg-blue-50'
                        }`}
                      >
                        {certFile ? (
                          <>
                            <FileCheck size={20} className="text-green-600" />
                            <div className="text-left">
                              <p className="text-sm font-medium text-green-700">{certFile.name}</p>
                              <p className="text-xs text-green-600">{(certFile.size / 1024).toFixed(1)} KB</p>
                            </div>
                          </>
                        ) : (
                          <>
                            <Upload size={20} className="text-gray-400" />
                            <div className="text-left">
                              <p className="text-sm text-gray-600">Click to upload certificate</p>
                              <p className="text-xs text-gray-400">fullchain.pem or .crt file</p>
                            </div>
                          </>
                        )}
                      </button>
                    </div>

                    <div>
                      <label className="block text-sm font-medium mb-1">Private Key File (.pem, .key)</label>
                      <input
                        ref={keyInputRef}
                        type="file"
                        accept=".pem,.key"
                        onChange={(e) => setKeyFile(e.target.files?.[0] ?? null)}
                        className="hidden"
                      />
                      <button
                        type="button"
                        onClick={() => keyInputRef.current?.click()}
                        className={`w-full flex items-center gap-3 px-4 py-3 border-2 border-dashed rounded-lg transition-colors ${
                          keyFile ? 'border-green-300 bg-green-50' : 'border-gray-300 hover:border-blue-400 hover:bg-blue-50'
                        }`}
                      >
                        {keyFile ? (
                          <>
                            <FileCheck size={20} className="text-green-600" />
                            <div className="text-left">
                              <p className="text-sm font-medium text-green-700">{keyFile.name}</p>
                              <p className="text-xs text-green-600">{(keyFile.size / 1024).toFixed(1)} KB</p>
                            </div>
                          </>
                        ) : (
                          <>
                            <Upload size={20} className="text-gray-400" />
                            <div className="text-left">
                              <p className="text-sm text-gray-600">Click to upload private key</p>
                              <p className="text-xs text-gray-400">privkey.pem or .key file</p>
                            </div>
                          </>
                        )}
                      </button>
                    </div>
                  </>
                ) : (
                  <>
                    <div>
                      <label className="block text-sm font-medium mb-1">Certificate (PEM)</label>
                      <textarea
                        value={certText}
                        onChange={(e) => setCertText(e.target.value)}
                        placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                        rows={6}
                        className="w-full px-3 py-2 border rounded-lg font-mono text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium mb-1">Private Key (PEM)</label>
                      <textarea
                        value={keyText}
                        onChange={(e) => setKeyText(e.target.value)}
                        placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
                        rows={6}
                        className="w-full px-3 py-2 border rounded-lg font-mono text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                      />
                    </div>
                  </>
                )}
              </>
            )}

            <div className="flex justify-end gap-2 border-t pt-4 mt-4">
              <button
                type="button"
                onClick={() => { setCreatedCert(null); onClose() }}
                className="px-4 py-2 text-sm border rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={saving}
                className={`px-4 py-2 text-sm text-white rounded-lg disabled:opacity-50 ${
                  tab === 'self-signed' ? 'bg-purple-600 hover:bg-purple-700' : 'bg-blue-600 hover:bg-blue-700'
                }`}
              >
                {saving ? 'Creating...' : 'Create'}
              </button>
            </div>
          </form>
        </>
      )}
    </Offcanvas>
  )
}
