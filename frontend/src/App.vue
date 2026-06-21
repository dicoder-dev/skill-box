<script setup>
import ProjectsView from './views/ProjectsView.vue'
</script>

<template>
  <main class="app">
    <ProjectsView />
  </main>
</template>

<style>
:root {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  font-size: 14px;
  line-height: 1.45;
  color: #1a1a1a;
  background: #ffffff;
}
html, body, #app { margin: 0; padding: 0; min-height: 100%; background: #ffffff; }
.app { min-height: 100vh; }
</style>
