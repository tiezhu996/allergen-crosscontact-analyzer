import { onBeforeUnmount, ref } from 'vue'

export function useAssessmentPolling(reload: () => Promise<void>, hasActive: () => boolean, interval = 2500) {
  const polling = ref(false)
  let timer: number | undefined
  async function tick() { if (!polling.value) return; await reload(); if (hasActive()) timer = window.setTimeout(tick, interval); else stop() }
  function start() { if (polling.value) return; polling.value = true; timer = window.setTimeout(tick, interval) }
  function stop() { polling.value = false; if (timer) window.clearTimeout(timer); timer = undefined }
  onBeforeUnmount(stop)
  return { polling, start, stop }
}
