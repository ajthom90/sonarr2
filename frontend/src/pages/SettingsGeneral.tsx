import { useState, useEffect } from 'react'
import { useGeneralSettings, useUpdateGeneralSettings } from '../api/hooks'
import styles from './SettingsGeneral.module.css'

export function SettingsGeneral() {
  const { data: settings, isLoading } = useGeneralSettings()
  const updateSettings = useUpdateGeneralSettings()
  const [tvdbKey, setTvdbKey] = useState('')
  const [saved, setSaved] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (settings) {
      setTvdbKey(settings.tvdbApiKey)
    }
  }, [settings])

  function handleSaveTvdb() {
    updateSettings.mutate({ tvdbApiKey: tvdbKey }, {
      onSuccess: () => {
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      },
    })
  }

  function handleCopyApiKey() {
    if (settings) {
      navigator.clipboard.writeText(settings.apiKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  if (isLoading) return <div className={styles.page}>Loading...</div>

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>General Settings</h1>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Metadata Source</h2>
        <label className={styles.label}>
          TVDB API Key
          <div className={styles.inputRow}>
            <input
              type="text"
              value={tvdbKey}
              onChange={(e) => setTvdbKey(e.target.value)}
              className={styles.input}
              placeholder="Enter your TVDB API key"
            />
            <button
              onClick={handleSaveTvdb}
              className={styles.saveButton}
              disabled={updateSettings.isPending || tvdbKey === settings?.tvdbApiKey}
            >
              {saved ? 'Saved!' : updateSettings.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
          <span className={styles.hint}>
            Required to search and add series. Get a free key at thetvdb.com
          </span>
        </label>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Security</h2>
        <label className={styles.label}>
          API Key
          <div className={styles.inputRow}>
            <input
              type="text"
              value={settings?.apiKey ?? ''}
              readOnly
              className={styles.input}
            />
            <button onClick={handleCopyApiKey} className={styles.copyButton}>
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <span className={styles.hint}>
            Used by external applications like Prowlarr to connect to sonarr2
          </span>
        </label>

        <div className={styles.field}>
          <span className={styles.fieldLabel}>Authentication</span>
          <span className={styles.fieldValue}>{settings?.authMode ?? 'forms'}</span>
        </div>
      </section>
    </div>
  )
}
