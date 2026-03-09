export function useScreenSize(breakpoint = 1024) {
  const isLargeScreen = ref(true)

  function update() {
    isLargeScreen.value = window.innerWidth >= breakpoint
  }

  onMounted(() => {
    update()
    window.addEventListener('resize', update)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', update)
  })

  return { isLargeScreen }
}
