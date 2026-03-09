<script setup lang="ts">
import { backendUrl } from '~/utils/api'

const colorMode = useColorMode()
const isSidebarOpen = ref(false)
const { highlight } = useSyntaxHighlighter()

const activeTab = ref('general')
const configRaw = ref('{\n  "permissions": {\n    "allow": [],\n    "deny": []\n  }\n}')
const highlightedHtml = ref('')
const loading = ref(false)
const saving = ref(false)
const saveSuccess = ref(false)

const editorContainerRef = ref<HTMLElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

// --- Profiles ---
interface Profile {
  name: string
  env: Record<string, string>
}

interface ProfilesConfig {
  active: string
  profiles: Record<string, Profile>
}

const profilesConfig = ref<ProfilesConfig>({ active: '', profiles: {} })
const profilesLoading = ref(false)
const profilesSaving = ref(false)
const editingProfile = ref<string | null>(null)
const editName = ref('')
const editEnv = ref<{ key: string; value: string; visible: boolean }[]>([])
const showNewProfile = ref(false)
const newProfileId = ref('')
const newProfileName = ref('')

async function fetchProfiles() {
  profilesLoading.value = true
  try {
    const res = await fetch(backendUrl('/api/profiles'))
    if (res.ok) {
      profilesConfig.value = await res.json()
    }
  } catch (e) {
    console.error('Failed to fetch profiles', e)
  } finally {
    profilesLoading.value = false
  }
}

async function saveFullProfiles() {
  profilesSaving.value = true
  try {
    const res = await fetch(backendUrl('/api/profiles'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(profilesConfig.value)
    })
    if (res.ok) {
      saveSuccess.value = true
      setTimeout(() => { saveSuccess.value = false }, 3000)
    }
  } catch (e) {
    console.error('Failed to save profiles', e)
  } finally {
    profilesSaving.value = false
  }
}

async function setActiveProfile(id: string) {
  try {
    const res = await fetch(backendUrl('/api/profiles/active'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ active: id })
    })
    if (res.ok) {
      profilesConfig.value.active = id
    }
  } catch (e) {
    console.error('Failed to set active profile', e)
  }
}

function startEditProfile(id: string) {
  const profile = profilesConfig.value.profiles[id]
  if (!profile) return
  editingProfile.value = id
  editName.value = profile.name
  editEnv.value = Object.entries(profile.env).map(([key, value]) => ({ key, value, visible: false }))
}

function cancelEdit() {
  editingProfile.value = null
  showNewProfile.value = false
}

function addEnvVar() {
  editEnv.value.push({ key: '', value: '', visible: false })
}

function removeEnvVar(idx: number) {
  editEnv.value.splice(idx, 1)
}

function saveEditProfile() {
  if (!editingProfile.value) return
  const env: Record<string, string> = {}
  for (const entry of editEnv.value) {
    if (entry.key.trim()) {
      env[entry.key.trim()] = entry.value
    }
  }
  profilesConfig.value.profiles[editingProfile.value] = {
    name: editName.value || editingProfile.value,
    env
  }
  editingProfile.value = null
  saveFullProfiles()
}

function createProfile() {
  const id = newProfileId.value.trim().toLowerCase().replace(/[^a-z0-9_-]/g, '')
  if (!id || profilesConfig.value.profiles[id]) return
  profilesConfig.value.profiles[id] = {
    name: newProfileName.value.trim() || id,
    env: {}
  }
  showNewProfile.value = false
  newProfileId.value = ''
  newProfileName.value = ''
  saveFullProfiles()
}

function deleteProfile(id: string) {
  if (profilesConfig.value.active === id) return
  if (!confirm(`Delete profile "${profilesConfig.value.profiles[id]?.name || id}"?`)) return
  delete profilesConfig.value.profiles[id]
  saveFullProfiles()
}

const profileEntries = computed(() => Object.entries(profilesConfig.value.profiles))

// --- Config ---
async function fetchConfig() {
  loading.value = true
  try {
    const res = await fetch(backendUrl('/config'))
    if (res.ok) {
      const data = await res.json()
      configRaw.value = JSON.stringify(data, null, 2)
    }
  } catch (e) {
    console.error('Failed to fetch config', e)
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  saveSuccess.value = false
  try {
    const parsed = JSON.parse(configRaw.value)
    const res = await fetch(backendUrl('/config'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(parsed)
    })

    if (res.ok) {
      saveSuccess.value = true
      configRaw.value = JSON.stringify(parsed, null, 2)
      setTimeout(() => { saveSuccess.value = false }, 3000)
    }
  } catch (e) {
    alert('Invalid JSON format! Please check your syntax.')
  } finally {
    saving.value = false
  }
}

function formatJson() {
  try {
    const parsed = JSON.parse(configRaw.value)
    configRaw.value = JSON.stringify(parsed, null, 2)
  } catch (e) {
    alert('Cannot format invalid JSON')
  }
}

function handleScroll(e: Event) {
  const textarea = e.target as HTMLTextAreaElement
  if (editorContainerRef.value) {
    editorContainerRef.value.scrollTop = textarea.scrollTop
    editorContainerRef.value.scrollLeft = textarea.scrollLeft
  }
}

watch(configRaw, async (newVal) => {
  highlightedHtml.value = await highlight(newVal, 'settings.json')
}, { immediate: true })

onMounted(() => {
  fetchConfig()
  fetchProfiles()
})

const tabs = [
  { id: 'general', label: 'Appearance', icon: 'i-lucide-palette' },
  { id: 'claude', label: 'Claude Code', icon: 'i-lucide-terminal' },
  { id: 'profiles', label: 'Profiles', icon: 'i-lucide-users' },
]
</script>

<template>
  <div class="h-full w-full overflow-hidden flex font-sans bg-background text-foreground">

    <USlideover
      v-model="isSidebarOpen"
      side="left"
      :ui="{ width: 'max-w-64', wrapper: 'z-[100]' }"
    >
      <Sidebar @navigate="isSidebarOpen = false" class="h-full" />
    </USlideover>

    <div class="flex-1 flex flex-col bg-background overflow-hidden">
      <div class="h-14 border-b border-border/40 flex items-center px-4 gap-3 shrink-0">
        <button
          @click="isSidebarOpen = true"
          class="w-9 h-9 rounded-lg flex items-center justify-center text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-900 hover:text-slate-700 dark:hover:text-slate-100 transition-colors shrink-0"
        >
          <UIcon name="i-lucide-menu" class="w-5 h-5" />
        </button>
        <h2 class="text-sm font-heading font-bold text-slate-900 dark:text-slate-100 tracking-tight">Settings</h2>
        <div v-if="loading" class="flex items-center gap-2 text-xs text-slate-400">
          <UIcon name="i-lucide-refresh-cw" class="w-3.5 h-3.5 animate-spin" />
          Loading...
        </div>
      </div>

      <div class="flex-1 p-6 sm:p-10 overflow-y-auto">
      <div class="max-w-4xl mx-auto space-y-6">
        <div class="border-b border-slate-200 dark:border-border/20 pb-4 space-y-4">
          <div class="flex bg-slate-100/50 dark:bg-slate-800/40 p-0.5 rounded-lg border border-slate-200 dark:border-border/10 shadow-sm w-fit">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              @click="activeTab = tab.id"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[13px] font-medium transition-all"
              :class="activeTab === tab.id
                ? 'bg-white dark:bg-slate-700/80 text-sky-500 dark:text-sky-400 shadow-sm'
                : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'"
            >
              <UIcon :name="tab.icon" class="w-3.5 h-3.5" />
              {{ tab.label }}
            </button>
          </div>
        </div>

        <Transition name="fade-slide" mode="out-in">
          <!-- GENERAL TAB -->
          <div v-if="activeTab === 'general'" class="space-y-8">
            <div class="space-y-4">
              <h3 class="text-xs font-bold uppercase tracking-widest text-slate-400 dark:text-slate-500 pb-1">Appearance</h3>
              <div class="p-5 bg-slate-50 dark:bg-slate-900/50 rounded-2xl border border-slate-200 dark:border-border/20 shadow-sm">
                <div class="flex items-center justify-between">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-sm font-semibold text-foreground">Theme</span>
                    <span class="text-xs text-slate-500">Personalize how the application looks</span>
                  </div>
                  <div class="flex bg-slate-100 dark:bg-slate-800 p-0.5 rounded-lg border border-slate-200 dark:border-border/20">
                    <button
                      v-for="opt in [
                        { value: 'system', label: 'System', icon: 'i-lucide-monitor' },
                        { value: 'light', label: 'Light', icon: 'i-lucide-sun' },
                        { value: 'dark', label: 'Dark', icon: 'i-lucide-moon' },
                      ]"
                      :key="opt.value"
                      @click="colorMode.preference = opt.value"
                      class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors"
                      :class="colorMode.preference === opt.value
                        ? 'bg-white dark:bg-slate-700 text-sky-500 dark:text-sky-400 shadow-sm'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'"
                    >
                      <UIcon :name="opt.icon" class="w-3.5 h-3.5" />
                      {{ opt.label }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- CLAUDE CODE TAB -->
          <div v-else-if="activeTab === 'claude'" class="space-y-6">
            <div class="flex items-center justify-between">
              <div class="space-y-1">
                <h3 class="text-xs font-bold uppercase tracking-widest text-slate-400 dark:text-slate-500">Claude Code Settings</h3>
                <p class="text-xs text-slate-500 flex items-center gap-1">
                   <UIcon name="i-lucide-file-text" class="w-3.5 h-3.5 text-slate-400" />
                   Editing ~/.claude/settings.json (User Scope)
                </p>
              </div>
              <div class="flex items-center gap-3">
                <button
                  @click="formatJson"
                  class="flex items-center gap-2 px-3 py-1.5 text-slate-500 hover:text-sky-500 dark:text-slate-400 dark:hover:text-sky-400 text-xs font-bold rounded-lg border border-slate-200 dark:border-border/20 transition-all bg-white dark:bg-slate-800"
                >
                  <UIcon name="i-lucide-sparkles" class="w-3.5 h-3.5" />
                  Format
                </button>
                <button
                  @click="saveConfig"
                  :disabled="saving"
                  class="flex items-center gap-2 px-5 py-2 bg-sky-500 hover:bg-sky-600 disabled:opacity-50 text-white text-xs font-bold rounded-xl transition-all shadow-lg shadow-sky-500/20"
                >
                  <UIcon v-if="saving" name="i-lucide-refresh-cw" class="w-3.5 h-3.5 animate-spin" />
                  <UIcon v-else name="i-lucide-cloud-upload" class="w-3.5 h-3.5" />
                  {{ saving ? 'Saving...' : 'Save All' }}
                </button>
              </div>
            </div>

            <div class="relative w-full h-[600px] bg-slate-50 dark:bg-slate-900/80 rounded-2xl border border-slate-200 dark:border-border/40 overflow-hidden group shadow-inner">
               <div class="absolute -top-3 left-4 px-2 bg-background text-[10px] font-bold text-slate-400 z-20 uppercase tracking-widest">JSON configuration</div>

               <!-- Syntax Highlight Layer -->
               <div
                 ref="editorContainerRef"
                 class="absolute inset-0 p-6 pointer-events-none overflow-hidden font-mono text-sm leading-relaxed whitespace-pre"
                 v-html="highlightedHtml"
               ></div>

               <!-- Editable Layer -->
               <textarea
                 ref="textareaRef"
                 v-model="configRaw"
                 @scroll="handleScroll"
                 spellcheck="false"
                 class="absolute inset-0 w-full h-full p-6 bg-transparent text-transparent caret-slate-900 dark:caret-white font-mono text-sm leading-relaxed outline-none resize-none whitespace-pre overflow-auto z-10"
                 placeholder='{ "permissions": { "allow": [] } }'
               ></textarea>
            </div>

            <div class="p-4 bg-sky-500/5 border border-sky-500/20 rounded-2xl flex gap-3">
               <UIcon name="i-lucide-info" class="w-5 h-5 text-sky-500 flex-shrink-0" />
               <div class="space-y-1">
                  <span class="text-xs font-bold text-sky-600 dark:text-sky-400 uppercase tracking-tight">Configuration Scope</span>
                  <p class="text-[11px] text-sky-600/70 dark:text-sky-400/60 leading-relaxed">
                     These settings are stored in <code>~/.claude/settings.json</code> and apply globally. You can define <code>permissions</code>, <code>env</code>, <code>model</code>, and more according to the official documentation.
                  </p>
               </div>
            </div>
          </div>

          <!-- PROFILES TAB -->
          <div v-else-if="activeTab === 'profiles'" class="space-y-6">
            <div class="flex items-center justify-between">
              <div class="space-y-1">
                <h3 class="text-xs font-bold uppercase tracking-widest text-slate-400 dark:text-slate-500">Profiles</h3>
                <p class="text-xs text-slate-500 flex items-center gap-1">
                  <UIcon name="i-lucide-file-text" class="w-3.5 h-3.5 text-slate-400" />
                  Manage environment profiles for Claude
                </p>
              </div>
              <button
                @click="showNewProfile = true"
                class="flex items-center gap-2 px-4 py-2 bg-sky-500 hover:bg-sky-600 text-white text-xs font-bold rounded-xl transition-all shadow-lg shadow-sky-500/20"
              >
                <UIcon name="i-lucide-plus" class="w-3.5 h-3.5" />
                New Profile
              </button>
            </div>

            <div v-if="profilesLoading" class="flex items-center justify-center py-12">
              <UIcon name="i-lucide-refresh-cw" class="w-5 h-5 text-slate-500 animate-spin" />
            </div>

            <template v-else>
              <!-- New profile form -->
              <div v-if="showNewProfile" class="p-5 bg-sky-500/5 border-2 border-sky-500/30 rounded-2xl space-y-4">
                <div class="flex items-center gap-2 text-sky-400">
                  <UIcon name="i-lucide-plus-circle" class="w-4 h-4" />
                  <span class="text-xs font-bold uppercase tracking-wider">New Profile</span>
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500 px-1">ID (slug)</label>
                    <input
                      v-model="newProfileId"
                      type="text"
                      placeholder="my-profile"
                      class="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-border/40 bg-white dark:bg-slate-900 text-sm text-foreground outline-none focus:border-sky-500/50"
                    />
                  </div>
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500 px-1">Display Name</label>
                    <input
                      v-model="newProfileName"
                      type="text"
                      placeholder="My Profile"
                      class="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-border/40 bg-white dark:bg-slate-900 text-sm text-foreground outline-none focus:border-sky-500/50"
                    />
                  </div>
                </div>
                <div class="flex gap-2">
                  <button
                    @click="createProfile"
                    :disabled="!newProfileId.trim()"
                    class="px-4 py-1.5 bg-sky-500 hover:bg-sky-600 disabled:opacity-50 text-white text-xs font-bold rounded-lg transition-all"
                  >
                    Create
                  </button>
                  <button
                    @click="cancelEdit"
                    class="px-4 py-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 text-xs font-bold rounded-lg border border-slate-200 dark:border-border/30 transition-all"
                  >
                    Cancel
                  </button>
                </div>
              </div>

              <!-- Profile list -->
              <div class="space-y-3">
                <div
                  v-for="[id, profile] in profileEntries"
                  :key="id"
                  class="p-5 rounded-2xl border transition-all"
                  :class="profilesConfig.active === id
                    ? 'bg-sky-500/5 border-sky-500/30 shadow-sm'
                    : 'bg-slate-50 dark:bg-slate-900/50 border-slate-200 dark:border-border/20'"
                >
                  <!-- View mode -->
                  <div v-if="editingProfile !== id">
                    <div class="flex items-center justify-between">
                      <div class="flex items-center gap-3">
                        <button
                          @click="setActiveProfile(id)"
                          class="w-5 h-5 rounded-full border-2 flex items-center justify-center transition-all"
                          :class="profilesConfig.active === id
                            ? 'border-sky-500 bg-sky-500'
                            : 'border-slate-300 dark:border-slate-600 hover:border-sky-400'"
                        >
                          <div v-if="profilesConfig.active === id" class="w-2 h-2 rounded-full bg-white" />
                        </button>
                        <div>
                          <div class="flex items-center gap-2">
                            <span class="text-sm font-semibold text-foreground">{{ profile.name }}</span>
                            <span class="text-[10px] font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded">{{ id }}</span>
                            <span v-if="profilesConfig.active === id" class="text-[10px] font-bold uppercase tracking-wider text-sky-500 bg-sky-500/10 px-1.5 py-0.5 rounded">Active</span>
                          </div>
                          <div class="text-xs text-slate-500 mt-0.5">
                            {{ Object.keys(profile.env).length }} env variable{{ Object.keys(profile.env).length !== 1 ? 's' : '' }}
                          </div>
                        </div>
                      </div>
                      <div class="flex items-center gap-1.5">
                        <button
                          @click="startEditProfile(id)"
                          class="p-1.5 rounded-lg text-slate-400 hover:text-sky-500 hover:bg-sky-500/10 transition-all"
                          title="Edit"
                        >
                          <UIcon name="i-lucide-pencil" class="w-3.5 h-3.5" />
                        </button>
                        <button
                          v-if="profilesConfig.active !== id"
                          @click="deleteProfile(id)"
                          class="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 transition-all"
                          title="Delete"
                        >
                          <UIcon name="i-lucide-trash-2" class="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  </div>

                  <!-- Edit mode -->
                  <div v-else class="space-y-4">
                    <div class="flex items-center gap-2 text-sky-400">
                      <UIcon name="i-lucide-pencil" class="w-4 h-4" />
                      <span class="text-xs font-bold uppercase tracking-wider">Editing "{{ id }}"</span>
                    </div>

                    <div class="space-y-1">
                      <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500 px-1">Display Name</label>
                      <input
                        v-model="editName"
                        type="text"
                        class="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-border/40 bg-white dark:bg-slate-900 text-sm text-foreground outline-none focus:border-sky-500/50"
                      />
                    </div>

                    <div class="space-y-2">
                      <div class="flex items-center justify-between">
                        <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500 px-1">Environment Variables</label>
                        <button
                          @click="addEnvVar"
                          class="flex items-center gap-1 text-[10px] font-bold text-sky-500 hover:text-sky-400 transition-colors"
                        >
                          <UIcon name="i-lucide-plus" class="w-3 h-3" />
                          Add Variable
                        </button>
                      </div>

                      <div v-for="(entry, idx) in editEnv" :key="idx" class="flex items-center gap-2">
                        <input
                          v-model="entry.key"
                          type="text"
                          placeholder="KEY"
                          class="flex-1 px-3 py-2 rounded-lg border border-slate-200 dark:border-border/40 bg-white dark:bg-slate-900 text-xs font-mono text-foreground outline-none focus:border-sky-500/50"
                        />
                        <div class="relative flex-[2]">
                          <input
                            v-model="entry.value"
                            :type="entry.visible ? 'text' : 'password'"
                            placeholder="value"
                            class="w-full px-3 py-2 pr-9 rounded-lg border border-slate-200 dark:border-border/40 bg-white dark:bg-slate-900 text-xs font-mono text-foreground outline-none focus:border-sky-500/50"
                          />
                          <button
                            @click="entry.visible = !entry.visible"
                            class="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                          >
                            <UIcon :name="entry.visible ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="w-3.5 h-3.5" />
                          </button>
                        </div>
                        <button
                          @click="removeEnvVar(idx)"
                          class="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 transition-all"
                        >
                          <UIcon name="i-lucide-x" class="w-3.5 h-3.5" />
                        </button>
                      </div>

                      <div v-if="editEnv.length === 0" class="text-xs text-slate-400 italic py-2 px-1">
                        No environment variables configured
                      </div>
                    </div>

                    <div class="flex gap-2 pt-2">
                      <button
                        @click="saveEditProfile"
                        class="px-4 py-1.5 bg-sky-500 hover:bg-sky-600 text-white text-xs font-bold rounded-lg transition-all"
                      >
                        Save
                      </button>
                      <button
                        @click="cancelEdit"
                        class="px-4 py-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 text-xs font-bold rounded-lg border border-slate-200 dark:border-border/30 transition-all"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <div class="p-4 bg-violet-500/5 border border-violet-500/20 rounded-2xl flex gap-3">
                <UIcon name="i-lucide-info" class="w-5 h-5 text-violet-500 flex-shrink-0" />
                <div class="space-y-1">
                  <span class="text-xs font-bold text-violet-600 dark:text-violet-400 uppercase tracking-tight">Profiles</span>
                  <p class="text-[11px] text-violet-600/70 dark:text-violet-400/60 leading-relaxed">
                    Profiles let you switch between different API configurations. Each profile defines environment variables that are passed to the Claude process. Stored in <code>~/.clauductor/profiles.json</code>.
                  </p>
                </div>
              </div>
            </template>
          </div>
        </Transition>
      </div>

      <!-- Toast Notification -->
      <Transition name="toast">
        <div v-if="saveSuccess" class="fixed bottom-10 right-10 bg-emerald-500 text-white px-6 py-3 rounded-2xl shadow-2xl flex items-center gap-3 z-[100] border border-white/20">
           <UIcon name="i-lucide-check-circle" class="w-6 h-6" />
           <div>
              <p class="text-sm font-bold">Settings saved!</p>
              <p class="text-[10px] opacity-80 uppercase tracking-widest font-bold">Configuration updated</p>
           </div>
        </div>
      </Transition>
    </div>
    </div>

  </div>
</template>

<style scoped>
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.25s ease-out;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-12px);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: scale(0.8) translateY(40px);
}

textarea {
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.2) transparent;
}

:deep(pre) {
  margin: 0;
  padding: 0 !important;
  background: transparent !important;
}

:deep(code) {
  font-family: inherit;
}
</style>
