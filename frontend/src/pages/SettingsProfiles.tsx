import { useState, useEffect } from 'react'
import {
  useQualityProfiles,
  useQualityDefinitions,
  useCreateQualityProfile,
  useUpdateQualityProfile,
  useDeleteQualityProfile,
} from '../api/hooks'
import { Modal } from '../components/Modal'
import type { QualityProfile, QualityDefinition, QualityProfileItem } from '../api/types'
import styles from './SettingsProfiles.module.css'

// ── Helpers ──────────────────────────────────────────────────────────────────

function qualityName(qualityId: number, definitions: QualityDefinition[]): string {
  const def = definitions.find((d) => d.id === qualityId)
  return def?.name ?? `Quality #${qualityId}`
}

// ── Profile form modal ──────────────────────────────────────────────────────

interface FormState {
  name: string
  upgradeAllowed: boolean
  cutoff: number
  items: Map<number, boolean> // qualityId -> allowed
}

function defaultFormState(definitions: QualityDefinition[]): FormState {
  const items = new Map<number, boolean>()
  for (const def of definitions) {
    items.set(def.id, false)
  }
  return {
    name: '',
    upgradeAllowed: false,
    cutoff: definitions.length > 0 ? definitions[0].id : 0,
    items,
  }
}

function profileToFormState(
  profile: QualityProfile,
  definitions: QualityDefinition[],
): FormState {
  const items = new Map<number, boolean>()
  // Initialize all definitions as not allowed
  for (const def of definitions) {
    items.set(def.id, false)
  }
  // Then set allowed from profile items
  for (const item of profile.items) {
    items.set(item.quality.id, item.allowed)
  }
  return {
    name: profile.name,
    upgradeAllowed: profile.upgradeAllowed,
    cutoff: profile.cutoff,
    items,
  }
}

interface ProfileFormModalProps {
  open: boolean
  onClose: () => void
  initial?: QualityProfile
  definitions: QualityDefinition[]
}

function ProfileFormModal({ open, onClose, initial, definitions }: ProfileFormModalProps) {
  const createMutation = useCreateQualityProfile()
  const updateMutation = useUpdateQualityProfile()
  const isEdit = !!initial

  const [form, setForm] = useState<FormState>(() =>
    initial ? profileToFormState(initial, definitions) : defaultFormState(definitions),
  )
  const [error, setError] = useState<string | null>(null)

  // Reset form when modal opens with different initial
  useEffect(() => {
    if (open) {
      setForm(
        initial ? profileToFormState(initial, definitions) : defaultFormState(definitions),
      )
      setError(null)
    }
  }, [open, initial, definitions])

  function handleClose() {
    onClose()
  }

  function handleToggleQuality(qualityId: number) {
    setForm((prev) => {
      const next = new Map(prev.items)
      next.set(qualityId, !next.get(qualityId))
      return { ...prev, items: next }
    })
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!form.name.trim()) {
      setError('Name is required')
      return
    }

    const items: Array<{ qualityId: number; allowed: boolean }> = []
    for (const [qualityId, allowed] of form.items) {
      items.push({ qualityId, allowed })
    }

    const body = {
      name: form.name.trim(),
      upgradeAllowed: form.upgradeAllowed,
      cutoff: form.cutoff,
      items,
      minFormatScore: 0,
      cutoffFormatScore: 0,
      formatItems: [] as unknown[],
    }

    if (isEdit && initial) {
      updateMutation.mutate(
        { id: initial.id, ...body },
        {
          onSuccess: handleClose,
          onError: (err) =>
            setError(err instanceof Error ? err.message : 'Failed to save'),
        },
      )
    } else {
      createMutation.mutate(body, {
        onSuccess: handleClose,
        onError: (err) =>
          setError(err instanceof Error ? err.message : 'Failed to create'),
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  // Only show allowed qualities as cutoff options
  const cutoffOptions = definitions.filter((d) => form.items.get(d.id))

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEdit ? 'Edit Quality Profile' : 'New Quality Profile'}
    >
      <form onSubmit={handleSubmit}>
        <div className={styles.field}>
          <label className={styles.label}>Name</label>
          <input
            className={styles.input}
            type="text"
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="Profile name"
          />
        </div>

        <div className={styles.field}>
          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={form.upgradeAllowed}
              onChange={(e) =>
                setForm((f) => ({ ...f, upgradeAllowed: e.target.checked }))
              }
              className={styles.checkbox}
            />
            Upgrade Allowed
          </label>
        </div>

        {form.upgradeAllowed && (
          <div className={styles.field}>
            <label className={styles.label}>Cutoff</label>
            <select
              className={styles.select}
              value={form.cutoff}
              onChange={(e) =>
                setForm((f) => ({ ...f, cutoff: Number(e.target.value) }))
              }
            >
              {cutoffOptions.length === 0 && (
                <option value={0}>Select allowed qualities first</option>
              )}
              {cutoffOptions.map((def) => (
                <option key={def.id} value={def.id}>
                  {def.name}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className={styles.qualitiesSection}>
          <div className={styles.qualitiesTitle}>Qualities</div>
          <div className={styles.qualityList}>
            {definitions.map((def) => (
              <div key={def.id} className={styles.qualityItem}>
                <input
                  type="checkbox"
                  checked={form.items.get(def.id) ?? false}
                  onChange={() => handleToggleQuality(def.id)}
                  className={styles.checkbox}
                />
                <span className={styles.qualityName}>{def.name}</span>
                <span className={styles.qualityMeta}>
                  {def.source} {def.resolution}
                </span>
              </div>
            ))}
            {definitions.length === 0 && (
              <div className={styles.qualityItem}>
                <span className={styles.muted}>No quality definitions available</span>
              </div>
            )}
          </div>
        </div>

        {error && <p className={styles.error}>{error}</p>}

        <div className={styles.actions}>
          <button type="submit" className={styles.saveBtn} disabled={isPending}>
            {isPending ? 'Saving...' : 'Save'}
          </button>
          <button type="button" className={styles.cancelBtn} onClick={handleClose}>
            Cancel
          </button>
        </div>
      </form>
    </Modal>
  )
}

// ── Profile card ────────────────────────────────────────────────────────────

interface ProfileCardProps {
  profile: QualityProfile
  definitions: QualityDefinition[]
  onEdit: (profile: QualityProfile) => void
  onDelete: (id: number) => void
  deleteDisabled: boolean
}

function ProfileCard({ profile, definitions, onEdit, onDelete, deleteDisabled }: ProfileCardProps) {
  const allowedQualities = profile.items.filter((i) => i.allowed)

  return (
    <div className={styles.card}>
      <div className={styles.cardHeader}>
        <h3 className={styles.cardName}>{profile.name}</h3>
        <div className={styles.cardActions}>
          <button className={styles.editBtn} onClick={() => onEdit(profile)}>
            Edit
          </button>
          <button
            className={styles.deleteBtn}
            onClick={() => onDelete(profile.id)}
            disabled={deleteDisabled}
          >
            Delete
          </button>
        </div>
      </div>

      <div className={styles.detailRow}>
        <span className={styles.detailLabel}>Upgrade Allowed:</span>
        <span
          className={`${styles.pill} ${profile.upgradeAllowed ? styles.pillEnabled : styles.pillDisabled}`}
        >
          {profile.upgradeAllowed ? 'Yes' : 'No'}
        </span>
      </div>

      {profile.upgradeAllowed && (
        <div className={styles.detailRow}>
          <span className={styles.detailLabel}>Cutoff:</span>
          <span className={styles.detailValue}>
            {qualityName(profile.cutoff, definitions)}
          </span>
        </div>
      )}

      <div className={styles.detailRow}>
        <span className={styles.detailLabel}>Allowed:</span>
        <span className={styles.detailValue}>
          {allowedQualities.length} qualities
        </span>
      </div>

      {allowedQualities.length > 0 && (
        <div className={styles.qualityTags}>
          {allowedQualities.map((item) => (
            <span
              key={item.quality.id}
              className={`${styles.qualityTag} ${item.quality.id === profile.cutoff ? styles.qualityTagCutoff : ''}`}
            >
              {qualityName(item.quality.id, definitions)}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

// ── Main page ───────────────────────────────────────────────────────────────

export function SettingsProfiles() {
  const { data: profilesData, isLoading, isError, error } = useQualityProfiles()
  const { data: definitions } = useQualityDefinitions()
  const deleteMutation = useDeleteQualityProfile()

  const [showForm, setShowForm] = useState(false)
  const [editTarget, setEditTarget] = useState<QualityProfile | undefined>(undefined)

  const profiles: QualityProfile[] = profilesData?.data ?? []
  const defs: QualityDefinition[] = definitions ?? []

  function openCreate() {
    setEditTarget(undefined)
    setShowForm(true)
  }

  function openEdit(profile: QualityProfile) {
    setEditTarget(profile)
    setShowForm(true)
  }

  function handleDelete(id: number) {
    if (confirm('Are you sure you want to delete this quality profile?')) {
      deleteMutation.mutate(id)
    }
  }

  if (isLoading) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Quality Profiles</h1>
        <p className={styles.stateMessage}>Loading quality profiles...</p>
      </div>
    )
  }

  if (isError) {
    return (
      <div className={styles.page}>
        <h1 className={styles.heading}>Quality Profiles</h1>
        <p className={styles.errorMessage}>
          Failed to load quality profiles:{' '}
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <h1 className={styles.heading}>Quality Profiles</h1>
        <button className={styles.addBtn} onClick={openCreate}>
          + New Profile
        </button>
      </div>

      {profiles.length === 0 ? (
        <p className={styles.stateMessage}>No quality profiles configured.</p>
      ) : (
        <div className={styles.grid}>
          {profiles.map((profile) => (
            <ProfileCard
              key={profile.id}
              profile={profile}
              definitions={defs}
              onEdit={openEdit}
              onDelete={handleDelete}
              deleteDisabled={deleteMutation.isPending}
            />
          ))}
        </div>
      )}

      <ProfileFormModal
        open={showForm}
        onClose={() => setShowForm(false)}
        initial={editTarget}
        definitions={defs}
      />
    </div>
  )
}
