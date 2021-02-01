export function sleep(sec) {
  return new Promise(res => {
    setTimeout(res, sec * 1000)
  })
}
