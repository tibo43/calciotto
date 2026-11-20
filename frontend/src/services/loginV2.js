// Which login page a signed-out browser gets. Design v2 is the default
// design now, so every browser gets the v2 page (LoginV2.vue). Read by
// App.vue (shell and fonts) and, through it, by LoginGate.vue.
export function shouldUseLoginV2() {
  return true;
}
