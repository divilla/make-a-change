<template>
  <q-layout view="hHh Lpr lFf">
    <q-header class="app-header">
      <q-toolbar class="app-toolbar">
        <q-toolbar-title class="app-title">AI Project Manager</q-toolbar-title>

        <div class="desktop-tabs-wrap">
          <q-tabs :model-value="activeTab" shrink stretch class="desktop-tabs">
            <q-route-tab name="home" to="/" label="Home" exact />
            <q-route-tab name="planning" to="/planning" label="Planning" />
            <q-route-tab name="projects" to="/projects" label="Projects" />
            <q-route-tab name="epics" to="/epics" label="Epics" />
            <q-route-tab name="changes" to="/changes" label="Changes" />
            <q-route-tab name="help" to="/help" label="Help" />
          </q-tabs>
        </div>

        <q-space />

        <q-toggle
          v-model="darkMode"
          dense
          color="grey-4"
          checked-icon="dark_mode"
          unchecked-icon="light_mode"
          class="top-dark-toggle"
          aria-label="Toggle dark mode"
        />

        <ProjectSelector dark :show-hint="false" class="top-project-selector" />

        <q-btn
          flat
          dense
          round
          icon="menu"
          class="mobile-menu"
          aria-label="Open navigation"
          @click="drawerOpen = true"
        />
      </q-toolbar>
    </q-header>

    <q-drawer v-model="drawerOpen" side="right" bordered :width="240">
      <q-list padding>
        <q-item clickable to="/" exact @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="dashboard" /></q-item-section>
          <q-item-section>Home</q-item-section>
        </q-item>
        <q-item clickable to="/planning" @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="psychology" /></q-item-section>
          <q-item-section>Planning</q-item-section>
        </q-item>
        <q-item clickable to="/projects" @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="view_kanban" /></q-item-section>
          <q-item-section>Projects</q-item-section>
        </q-item>
        <q-item clickable to="/epics" @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="view_timeline" /></q-item-section>
          <q-item-section>Epics</q-item-section>
        </q-item>
        <q-item clickable to="/changes" @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="published_with_changes" /></q-item-section>
          <q-item-section>Changes</q-item-section>
        </q-item>
        <q-item clickable to="/help" @click="drawerOpen = false">
          <q-item-section avatar><q-icon name="help" /></q-item-section>
          <q-item-section>Help</q-item-section>
        </q-item>
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useQuasar } from 'quasar';
import { useRoute, useRouter } from 'vue-router';
import ProjectSelector from '@/features/projects/components/ProjectSelector.vue';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import { refreshProjectScope } from '@/features/projects/services/projectScopeRefresh';
import {
  PROJECT_CHANGE_LOADING_PATH,
  projectChangeTargetPath,
} from '@/router/projectChangeRedirect';

const $q = useQuasar();
const route = useRoute();
const router = useRouter();
const drawerOpen = ref(false);
const darkMode = computed({
  get: () => $q.dark.isActive,
  set: (value) => {
    $q.dark.set(value);
  },
});
const projectSelection = useProjectSelectionStore();
const { projects, hasLoaded, currentProjectId } = storeToRefs(projectSelection);
const projectChangeRedirectReady = ref(false);
let projectSwitchToken = 0;

const activeTab = computed(() => {
  if (route.path.startsWith('/planning')) return 'planning';
  if (route.path.startsWith('/projects')) return 'projects';
  if (route.path.startsWith('/epics')) return 'epics';
  if (route.path.startsWith('/changes')) return 'changes';
  if (route.path.startsWith('/help')) return 'help';
  return 'home';
});

onMounted(() => {
  void projectSelection.loadProjects().catch(() => undefined);
});

watch(
  hasLoaded,
  async (loaded) => {
    if (!loaded) return;

    await nextTick();
    projectChangeRedirectReady.value = true;
  },
  { immediate: true },
);

watch(currentProjectId, () => {
  if (!projectChangeRedirectReady.value) return;

  const token = ++projectSwitchToken;
  const loadingTarget = typeof route.query.to === 'string' ? route.query.to : '';
  const targetPath =
    projectSelection.routeDrivenTargetPath ||
    (route.path === PROJECT_CHANGE_LOADING_PATH && loadingTarget
      ? loadingTarget
      : projectChangeTargetPath(route.path));

  void (async () => {
    projectSelection.setSwitchingProject(true);
    try {
      if (route.path !== PROJECT_CHANGE_LOADING_PATH) {
        await router.replace({
          path: PROJECT_CHANGE_LOADING_PATH,
          query: { to: targetPath },
        });
      }

      await refreshProjectScope().catch(() => undefined);
      if (token !== projectSwitchToken) return;

      const finalPath = projects.value.length ? targetPath : '/projects';
      await router.replace(finalPath);
    } finally {
      if (token === projectSwitchToken) {
        projectSelection.clearRouteDrivenProjectSwitch();
        projectSelection.setSwitchingProject(false);
      }
    }
  })();
});

watch([hasLoaded, projects, () => route.path], () => {
  if (
    hasLoaded.value &&
    !projects.value.length &&
    route.path !== '/projects' &&
    route.path !== PROJECT_CHANGE_LOADING_PATH
  ) {
    void router.push('/projects');
  }
});
</script>
