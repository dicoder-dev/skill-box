<script setup>
import { ref } from 'vue'
import ProjectsView from './views/ProjectsView.vue'
import SkillsView from './views/SkillsView.vue'
import MarketView from './views/MarketView.vue'

const tab = ref('skills')
</script>

<template>
  <main class="app">
    <nav class="topnav">
      <button :class="{ active: tab === 'skills' }" @click="tab = 'skills'">Skills</button>
      <button :class="{ active: tab === 'projects' }" @click="tab = 'projects'">Projects</button>
      <button :class="{ active: tab === 'market' }" @click="tab = 'market'">Market</button>
    </nav>
    <SkillsView v-if="tab === 'skills'" />
    <ProjectsView v-else-if="tab === 'projects'" />
    <MarketView v-else />
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
.topnav {
  display: flex;
  gap: 0;
  border-bottom: 1px solid #e5e7eb;
  padding: 0 20px;
  background: #f9fafb;
}
.topnav button {
  border: none;
  background: transparent;
  padding: 10px 16px;
  font-size: 14px;
  color: #6b7280;
  border-bottom: 2px solid transparent;
  cursor: pointer;
}
.topnav button.active {
  color: #2563eb;
  border-bottom-color: #2563eb;
  font-weight: 600;
}
</style>
