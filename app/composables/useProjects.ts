import type { ProjectInfo, SessionSummary } from '~/types'
import { backendUrl } from '~/utils/api'

export function useProjects() {
  const projects = ref<ProjectInfo[]>([])
  const sessions = ref<SessionSummary[]>([])
  const selectedProject = ref<ProjectInfo | null>(null)
  const loading = ref(false)

  async function fetchProjects() {
    loading.value = true
    try {
      const res = await fetch(backendUrl('/projects'))
      projects.value = await res.json()
    } finally {
      loading.value = false
    }
  }

  async function selectProject(project: ProjectInfo) {
    selectedProject.value = project
    loading.value = true
    try {
      const res = await fetch(backendUrl(`/projects/${project.encodedDir}/sessions`))
      sessions.value = await res.json()
    } finally {
      loading.value = false
    }
  }

  async function fetchSessionsForPath(cwdPath: string) {
    if (!cwdPath) {
      sessions.value = []
      return
    }
    const encoded = cwdPath.replace(/\//g, '-')
    loading.value = true
    try {
      const res = await fetch(backendUrl(`/projects/${encoded}/sessions`))
      if (res.ok) {
        sessions.value = await res.json()
      } else {
        sessions.value = []
      }
    } catch {
      sessions.value = []
    } finally {
      loading.value = false
    }
  }

  async function checkPathExists(path: string): Promise<boolean> {
    try {
      const res = await fetch(backendUrl('/projects/check'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path })
      })
      if (res.ok) {
        const data = await res.json()
        return data.exists
      }
    } catch (e) {
      console.error(e)
    }
    return false
  }

  async function createProjectFolder(path: string): Promise<boolean> {
    try {
      const res = await fetch(backendUrl('/projects/create'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path })
      })
      return res.ok
    } catch (e) {
      console.error(e)
    }
    return false
  }

  function clearProject() {
    selectedProject.value = null
    sessions.value = []
  }

  return {
    projects,
    sessions,
    selectedProject,
    loading,
    fetchProjects,
    selectProject,
    fetchSessionsForPath,
    checkPathExists,
    createProjectFolder,
    clearProject,
  }
}

