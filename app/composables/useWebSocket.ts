import type { BackendMessage } from '~/types'
import { wsUrl } from '~/utils/api'

export function useWebSocket() {
  const isConnected = ref(false)
  const clientId = ref<string | null>(null)

  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let messageHandler: ((msg: BackendMessage) => void) | null = null
  let reconnectHandler: ((newClientId: string) => void) | null = null
  let firstConnect = true

  function connect() {
    ws = new WebSocket(wsUrl('/ws'))

    ws.onopen = () => {
      isConnected.value = true
    }

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      if (msg.type === 'connected') {
        clientId.value = msg.clientId
        if (firstConnect) {
          firstConnect = false
        } else {
          reconnectHandler?.(msg.clientId)
        }
        return
      }
      messageHandler?.(msg)
    }

    ws.onclose = () => {
      isConnected.value = false
      reconnectTimer = setTimeout(connect, 2000)
    }
  }

  function onMessage(handler: (msg: BackendMessage) => void) {
    messageHandler = handler
  }

  function onReconnect(handler: (newClientId: string) => void) {
    reconnectHandler = handler
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    ws?.close()
    ws = null
  }

  onMounted(connect)
  onUnmounted(disconnect)

  return { isConnected, clientId, onMessage, onReconnect }
}
