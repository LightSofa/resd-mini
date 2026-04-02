import { defineStore } from 'pinia'
import { ref } from 'vue'
import type {wsType} from "@/types/ws"

export const useWsStore = defineStore('ws-store', () => {
    const websocketTask = ref()
    const heartbeatTime = ref()
    const isOpenSocket = ref(false)
    const isReconnect = ref(true)
    const port = ref("8899")
    const host = ref("")

    const onMessageHandles = ref<any>({
        ping: (res: any) => {
            console.log("websocket-ping", res)
        },
        opened: (res: any) => {
            console.log("websocket-opened", res)
        },
        close: (res: any) => {
            console.log("websocket-close", res)
        },
        error: (res: any) => {
            console.log("websocket-error", res)
        },
    })

    const setPort = (p: string)=>{
        port.value = p
    }

    const setHost = (h: string) => {
        host.value = h
    }

    const resolveWsHost = () => {
        const fromConfig = (host.value || "").replace(/^\[|\]$/g, "").trim()
        if (!fromConfig || fromConfig === "0.0.0.0" || fromConfig === "::") {
            return (typeof window !== "undefined" && window.location.hostname) ? window.location.hostname : "localhost"
        }
        return fromConfig
    }

    const websocketInit = () => {
        if (isOpenSocket.value) {
            return
        }
        let wsURL = "ws://" + resolveWsHost() + ":" + port.value + "/api/ws"
        if (typeof window !== "undefined" && /^https?:$/.test(window.location.protocol)) {
            const wsProto = window.location.protocol === "https:" ? "wss:" : "ws:"
            wsURL = `${wsProto}//${window.location.host}/api/ws`
        }

        websocketTask.value = new WebSocket(wsURL)
        websocketTask.value.onopen = (event: wsType.Event) => {
            heartbeatTime.value && clearInterval(heartbeatTime.value)
            isOpenSocket.value = true
            heartbeatTime.value = setInterval(() => {
                websocketTask.value.send(JSON.stringify({ type: "ping" }))
            }, 50000)
            onMessageHandles.value.opened(event)
        }

        websocketTask.value.onmessage = (event: wsType.Event) => {
            const data = JSON.parse(event.data)
            if (onMessageHandles.value.hasOwnProperty(data.type)) {
                onMessageHandles.value[data.type](data.data)
            } else {
                console.log("找不到该类型的消息处理器")
            }
        }

        websocketTask.value.onclose = () => {
            isOpenSocket.value = false
            if (isReconnect.value) reconnect()
            onMessageHandles.value.close()
        }

        websocketTask.value.onerror = (error: any) => {
            console.error("WebSocket 错误", error)
            onMessageHandles.value.error(error)
        }
    }

    const reconnect = () => {
        if (!isOpenSocket.value) {
            setTimeout(() => {
                websocketInit()
            }, 2000)
        }
    }

    const send = (data: any) => {
        if (isOpenSocket.value && websocketTask.value) {
            websocketTask.value.send(JSON.stringify(data))
        } else {
            console.warn("WebSocket 未连接")
        }
    }

    const close = () => {
        if (websocketTask.value) {
            websocketTask.value.close()
            isOpenSocket.value = false
            websocketTask.value = null
        }
    }

    const bindMessageHandle = (handle: wsType.Handle) => {
        onMessageHandles.value[handle.type] = handle.event
    }

    return {
        setHost, setPort, websocketInit, send, close, bindMessageHandle
    }
})
