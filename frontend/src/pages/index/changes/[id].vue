<template>
  <q-page class="app-page">
    <q-banner v-if="error" class="status-banner status-banner--error" rounded>
      <template #avatar>
        <q-icon name="warning" />
      </template>
      {{ error }}
    </q-banner>

    <div v-if="loading" class="empty-state">
      <q-spinner size="32px" color="primary" />
    </div>

    <div v-else-if="!currentChange" class="empty-state">
      <q-icon name="published_with_changes" size="44px" />
      <span>Change not found.</span>
    </div>

    <template v-else>
      <q-markup-table flat bordered class="change-detail-table">
        <thead>
          <tr>
            <th class="text-right">Ref</th>
            <th class="text-center">Types</th>
            <th class="text-left">Title</th>
            <th class="text-center">Epic</th>
            <th class="text-center">Phase</th>
            <th class="text-center">Open</th>
            <th class="text-center">Complete</th>
            <th class="text-center">Modified</th>
            <th class="text-center">Version</th>
            <th class="change-actions-cell"></th>
          </tr>
        </thead>
        <tbody>
          <tr class="bg-primary text-white text-weight-bold">
            <td class="text-right">{{ currentChange.ref ? `#${currentChange.ref}` : '' }}</td>
            <td class="text-center">{{ currentChange.change_types.join(', ') }}</td>
            <td class="text-left">
              <div>{{ currentChange.title }}</div>
              <div class="text-caption text-white">{{ currentChange.slug || '' }}</div>
            </td>
            <td class="text-center">{{ epicName(currentChange) }}</td>
            <td class="text-center">{{ currentChange.change_phase }}</td>
            <td class="text-center">{{ currentChange.open ? 'Yes' : 'No' }}</td>
            <td class="text-center">{{ completionLabel(currentChange) }}</td>
            <td class="text-center">{{ formatModified(currentChange.modified) }}</td>
            <td class="text-center">{{ currentChange.version }}</td>
            <td class="change-actions-cell">
              <q-btn-dropdown
                flat
                round
                color="white"
                dropdown-icon="more_vert"
                aria-label="Change actions"
                @click.stop
              >
                <q-list>
                  <q-item
                    clickable
                    v-close-popup
                    data-action="edit-change"
                    @click.stop="openChangeEdit(currentChange.id)"
                  >
                    <q-item-section>
                      <q-item-label>Edit</q-item-label>
                    </q-item-section>
                  </q-item>

                  <q-item
                    clickable
                    v-close-popup
                    data-action="delete-change"
                    :disable="isChangeDeleting(currentChange.id)"
                    @click.stop="confirmDeleteChange(currentChange)"
                  >
                    <q-item-section>
                      <q-item-label>Delete</q-item-label>
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-btn-dropdown>
            </td>
          </tr>

          <tr class="text-weight-bold">
            <td class="text-right">nr</td>
            <td class="text-center">&nbsp;</td>
            <td class="text-left" colspan="4">Test Case</td>
            <td class="text-center">Complete</td>
            <td class="text-center">Modified</td>
            <td class="text-center">Version</td>
            <td class="change-actions-cell"></td>
          </tr>
          <tr v-for="testCase in testCases" :key="testCase.id">
            <td class="text-right">#{{ testCase.id }}</td>
            <td class="text-center">&nbsp;</td>
            <td class="text-left" colspan="4">
              {{ testCase.scenario }}
            </td>
            <td class="text-center">
              <q-checkbox
                :model-value="testCase.done"
                dense
                color="secondary"
                :disable="isTestCaseUpdating(testCase.id)"
                @update:model-value="toggleTestCase(testCase, Boolean($event))"
              />
            </td>
            <td class="text-center">{{ formatModified(testCase.modified) }}</td>
            <td class="text-center">{{ testCase.version }}</td>
            <td class="change-actions-cell">
              <q-btn-dropdown
                flat
                round
                color="secondary"
                dropdown-icon="more_vert"
                aria-label="Test case actions"
                @click.stop
              >
                <q-list>
                  <q-item
                    clickable
                    v-close-popup
                    data-action="edit-test-case"
                    :disable="isTestCaseUpdating(testCase.id)"
                    @click.stop="openTestCaseEdit(testCase)"
                  >
                    <q-item-section>
                      <q-item-label>Edit</q-item-label>
                    </q-item-section>
                  </q-item>

                  <q-item
                    clickable
                    v-close-popup
                    data-action="delete-test-case"
                    :disable="isTestCaseUpdating(testCase.id)"
                    @click.stop="confirmDeleteTestCase(testCase)"
                  >
                    <q-item-section>
                      <q-item-label>Delete</q-item-label>
                    </q-item-section>
                  </q-item>
                </q-list>
              </q-btn-dropdown>
            </td>
          </tr>
          <tr>
            <td>&nbsp;</td>
            <td>&nbsp;</td>
            <td colspan="8">
              <q-btn
                rounded
                color="secondary"
                icon="library_add_check"
                label="Add Test Case"
                @click="openTestCaseCreate"
              />
            </td>
          </tr>
        </tbody>
      </q-markup-table>

      <q-markup-table flat bordered class="change-detail-table">
        <tbody>
          <tr>
            <td class="text-right text-weight-bold">Agent Edit</td>
            <td>{{ currentChange.agent_edit ? 'Yes' : 'No' }}</td>
          </tr>
          <tr>
            <td class="text-right text-weight-bold">Definition</td>
            <td class="change-detail-artifact-cell">
              <pre class="change-detail-text">{{ currentChange.def }}</pre>
            </td>
          </tr>
          <tr v-if="currentChange.pr_url">
            <td class="text-right text-weight-bold">PR URL</td>
            <td>
              <a v-if="safePRUrl" :href="safePRUrl" target="_blank" rel="noreferrer">
                {{ currentChange.pr_url }}
              </a>
              <span v-else>{{ currentChange.pr_url }}</span>
            </td>
          </tr>
          <tr v-if="currentChange.spec_html">
            <td class="change-detail-artifact-cell" style="padding-top: 32px; padding-bottom: 32px">
              <div class="apply-markdown" v-html="currentChange.spec_html" />
            </td>
          </tr>
          <tr v-if="currentChange.pr_html">
            <td class="change-detail-artifact-cell" style="padding-top: 32px; padding-bottom: 32px">
              <div class="apply-markdown" v-html="currentChange.pr_html" />
            </td>
          </tr>
        </tbody>
      </q-markup-table>
    </template>

    <q-dialog v-model="testCaseEditOpen">
      <q-card class="test-case-edit-dialog">
        <q-card-section>
          <div class="text-h6">Edit Test Case</div>
        </q-card-section>

        <q-form @submit.prevent="saveTestCaseEdit">
          <q-card-section class="test-case-edit-fields">
            <q-input
              v-model="editingTestCaseScenario"
              autofocus
              filled
              autogrow
              type="textarea"
              input-style="min-height: 72px"
              label="Test Case"
            />
            <q-select
              v-model="editingTestCaseChangeId"
              filled
              emit-value
              map-options
              label="Change"
              :options="changeOptions"
            />
          </q-card-section>

          <q-card-actions align="right">
            <q-btn
              flat
              icon="close"
              label="Cancel"
              :disable="savingTestCase"
              @click="closeTestCaseEdit"
            />
            <q-btn
              color="secondary"
              icon="save"
              label="Save"
              type="submit"
              :loading="savingTestCase"
              :disable="!editingTestCaseScenario.trim()"
            />
          </q-card-actions>
        </q-form>
      </q-card>
    </q-dialog>

    <q-dialog v-model="testCaseCreateOpen">
      <q-card class="test-case-edit-dialog">
        <q-card-section>
          <div class="text-h6">Add Test Case</div>
        </q-card-section>

        <q-form @submit.prevent="saveTestCaseCreate">
          <q-card-section>
            <q-input
              v-model="newTestCaseScenario"
              autofocus
              filled
              autogrow
              type="textarea"
              input-style="min-height: 72px"
              label="Test Case"
            />
          </q-card-section>

          <q-card-actions align="right">
            <q-btn
              flat
              icon="close"
              label="Cancel"
              :disable="creatingTestCase"
              @click="closeTestCaseCreate"
            />
            <q-btn
              color="secondary"
              icon="save"
              label="Add"
              type="submit"
              :loading="creatingTestCase"
              :disable="!newTestCaseScenario.trim()"
            />
          </q-card-actions>
        </q-form>
      </q-card>
    </q-dialog>

    <DeleteConfirmationDialog v-model:open="deleteConfirmationOpen" @confirm="confirm" />

    <q-dialog v-model="projectMismatchOpen" persistent>
      <q-card class="project-mismatch-dialog">
        <q-card-section>
          <div class="text-h6">Switch project?</div>
        </q-card-section>

        <q-card-section>
          This change belongs to {{ projectMismatch?.requiredProjectName }}. The selected project is
          {{ projectMismatch?.currentProjectName }}.
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Stay" @click="stayOnCurrentProject" />
          <q-btn color="primary" label="Switch" @click="switchToChangeProject" />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { storeToRefs } from 'pinia';
import { useRoute, useRouter } from 'vue-router';
import { deleteChange, getChange } from '@/features/changes/api/changeApi';
import { useChangeCacheStore } from '@/features/changes/model/changeCache.store';
import type { Change } from '@/features/changes/model/change.types';
import { useProjectSelectionStore } from '@/features/projects/model/projectSelection.store';
import {
  createTestCase,
  deleteTestCase,
  updateTestCase,
  updateTestCaseChange,
  updateTestCaseDone,
} from '@/features/test-cases/api/testCaseApi';
import type { TestCase } from '@/features/test-cases/model/testCase.types';
import DeleteConfirmationDialog from '@/shared/ui/DeleteConfirmationDialog.vue';

interface ProjectMismatch {
  currentProjectId: number;
  currentProjectName: string;
  requiredProjectId: number;
  requiredProjectName: string;
}

const route = useRoute();
const router = useRouter();
const projectSelection = useProjectSelectionStore();
const changeCache = useChangeCacheStore();
const { changes, epics } = storeToRefs(changeCache);

const testCases = ref<TestCase[]>([]);
const detailChange = ref<Change | null>(null);
const detailLoading = ref(false);
const deletingChangeIds = ref<number[]>([]);
const updatingTestCaseIds = ref<number[]>([]);
const testCaseCreateOpen = ref(false);
const newTestCaseScenario = ref('');
const creatingTestCase = ref(false);
const testCaseEditOpen = ref(false);
const editingTestCase = ref<TestCase | null>(null);
const editingTestCaseScenario = ref('');
const editingTestCaseChangeId = ref<number | null>(null);
const savingTestCase = ref(false);
const deleteConfirmationOpen = ref(false);
const pendingDeleteAction = ref<(() => Promise<void>) | null>(null);
const projectMismatchOpen = ref(false);
const projectMismatch = ref<ProjectMismatch | null>(null);
const testCaseError = ref('');
const error = computed(() => projectSelection.error || testCaseError.value);
const loading = computed(() => detailLoading.value || changeCache.loading);
const changeId = computed(() => Number(route.params.id));
const currentChange = computed(() =>
  detailChange.value?.id === changeId.value ? detailChange.value : null,
);
const safePRUrl = computed(() => normalizeHTTPURL(currentChange.value?.pr_url || ''));
const changeOptions = computed(() =>
  changes.value.map((change) => ({ label: `#${change.id} ${change.title}`, value: change.id })),
);

function normalizeHTTPURL(value: string) {
  try {
    const parsed = new URL(value);
    if (!parsed.hostname || (parsed.protocol !== 'https:' && parsed.protocol !== 'http:')) {
      return '';
    }
    return parsed.href;
  } catch {
    return '';
  }
}

async function ensureProjectsLoaded() {
  if (!projectSelection.hasLoaded) {
    await projectSelection.loadProjects();
  }
}

async function loadChangeDetail() {
  if (!changeId.value) return;

  detailLoading.value = true;
  testCaseError.value = '';
  try {
    const detail = await getChange(changeId.value);
    detailChange.value = detail.change;
    testCases.value = detail.test_cases;
    await ensureProjectsLoaded();
    detectProjectMismatch(detail.change);
    await changeCache.loadProjectChanges(detail.change.project_id);
    changeCache.upsertChange(detail.change);
  } catch (err) {
    detailChange.value = null;
    testCases.value = [];
    testCaseError.value = err instanceof Error ? err.message : 'Unable to load change.';
  } finally {
    detailLoading.value = false;
  }
}

function projectName(projectId: number) {
  return (
    projectSelection.projects.find((project) => project.id === projectId)?.name || `#${projectId}`
  );
}

function detectProjectMismatch(change: Change) {
  const currentProjectId = projectSelection.currentProjectId;
  if (
    !currentProjectId ||
    currentProjectId === change.project_id ||
    projectSelection.isSwitchingProject
  ) {
    projectMismatchOpen.value = false;
    projectMismatch.value = null;
    return;
  }

  projectMismatch.value = {
    currentProjectId,
    currentProjectName: projectName(currentProjectId),
    requiredProjectId: change.project_id,
    requiredProjectName: projectName(change.project_id),
  };
  projectMismatchOpen.value = true;
}

function switchToChangeProject() {
  const mismatch = projectMismatch.value;
  if (!mismatch) return;

  projectMismatchOpen.value = false;
  projectMismatch.value = null;
  projectSelection.beginRouteDrivenProjectSwitch(route.fullPath);
  projectSelection.selectProject(mismatch.requiredProjectId);
}

function stayOnCurrentProject() {
  projectMismatchOpen.value = false;
  projectMismatch.value = null;

  if (typeof window !== 'undefined' && window.history.length > 1) {
    router.back();
    return;
  }

  void router.replace('/changes');
}

function applyTestCaseMutation(testCaseList: TestCase[], change: Change) {
  if (change.id === changeId.value) {
    testCases.value = testCaseList;
    detailChange.value = change;
  }
  changeCache.upsertChange(change);
}

function isTestCaseUpdating(id: number) {
  return updatingTestCaseIds.value.includes(id);
}

function isChangeDeleting(id: number) {
  return deletingChangeIds.value.includes(id);
}

function confirmDeleteChange(change: Change) {
  if (isChangeDeleting(change.id)) return;
  pendingDeleteAction.value = () => deleteChangeConfirmed(change);
  deleteConfirmationOpen.value = true;
}

async function confirm() {
  const action = pendingDeleteAction.value;
  deleteConfirmationOpen.value = false;
  pendingDeleteAction.value = null;
  if (!action) return;

  await action();
}

async function deleteChangeConfirmed(change: Change) {
  if (!change || isChangeDeleting(change.id)) return;

  deletingChangeIds.value = [...deletingChangeIds.value, change.id];
  testCaseError.value = '';

  try {
    await deleteChange(change.id);
    await changeCache.loadProjectChanges(change.project_id);
    await projectSelection.loadProjects().catch(() => undefined);

    if (change.id === currentChange.value?.id) {
      void router.push('/changes');
    }
  } catch (err) {
    testCaseError.value = err instanceof Error ? err.message : 'Unable to delete change.';
  } finally {
    deletingChangeIds.value = deletingChangeIds.value.filter((id) => id !== change.id);
  }
}

function confirmDeleteTestCase(testCase: TestCase) {
  if (isTestCaseUpdating(testCase.id)) return;
  pendingDeleteAction.value = () => deleteTestCaseConfirmed(testCase);
  deleteConfirmationOpen.value = true;
}

async function deleteTestCaseConfirmed(testCase: TestCase) {
  if (isTestCaseUpdating(testCase.id)) return;

  updatingTestCaseIds.value = [...updatingTestCaseIds.value, testCase.id];
  testCaseError.value = '';

  try {
    const mutation = await deleteTestCase(testCase.id);
    applyTestCaseMutation(mutation.test_cases, mutation.change);
    await changeCache.loadProjectChanges(mutation.change.project_id);
    if (editingTestCase.value?.id === testCase.id) closeTestCaseEdit();
  } catch (err) {
    testCaseError.value = err instanceof Error ? err.message : 'Unable to delete test case.';
  } finally {
    updatingTestCaseIds.value = updatingTestCaseIds.value.filter((id) => id !== testCase.id);
  }
}

async function toggleTestCase(testCase: TestCase, done: boolean) {
  if (isTestCaseUpdating(testCase.id)) return;

  updatingTestCaseIds.value = [...updatingTestCaseIds.value, testCase.id];
  testCaseError.value = '';

  try {
    const mutation = await updateTestCaseDone(testCase.id, done);
    applyTestCaseMutation(mutation.test_cases, mutation.change);
    await changeCache.loadProjectChanges(mutation.change.project_id);
  } catch (err) {
    testCaseError.value = err instanceof Error ? err.message : 'Unable to update test case.';
  } finally {
    updatingTestCaseIds.value = updatingTestCaseIds.value.filter((id) => id !== testCase.id);
  }
}

function openTestCaseEdit(testCase: TestCase) {
  editingTestCase.value = testCase;
  editingTestCaseScenario.value = testCase.scenario;
  editingTestCaseChangeId.value = testCase.change_id;
  testCaseEditOpen.value = true;
}

function closeTestCaseEdit() {
  if (savingTestCase.value) return;

  testCaseEditOpen.value = false;
  editingTestCase.value = null;
  editingTestCaseScenario.value = '';
  editingTestCaseChangeId.value = null;
}

function openTestCaseCreate() {
  newTestCaseScenario.value = '';
  testCaseCreateOpen.value = true;
}

function closeTestCaseCreate() {
  if (creatingTestCase.value) return;

  testCaseCreateOpen.value = false;
  newTestCaseScenario.value = '';
}

async function saveTestCaseCreate() {
  if (!currentChange.value) return;

  const scenario = newTestCaseScenario.value.trim();
  if (!scenario) return;

  creatingTestCase.value = true;
  testCaseError.value = '';

  try {
    const mutation = await createTestCase(currentChange.value.id, scenario);
    applyTestCaseMutation(mutation.test_cases, mutation.change);
    testCaseCreateOpen.value = false;
    newTestCaseScenario.value = '';
    await changeCache.loadProjectChanges(mutation.change.project_id);
  } catch (err) {
    testCaseError.value = err instanceof Error ? err.message : 'Unable to create test case.';
  } finally {
    creatingTestCase.value = false;
  }
}

async function saveTestCaseEdit() {
  if (!editingTestCase.value) return;

  const scenario = editingTestCaseScenario.value.trim();
  if (!scenario) return;

  savingTestCase.value = true;
  testCaseError.value = '';

  try {
    let mutation = await updateTestCase({
      id: editingTestCase.value.id,
      scenario,
    });
    if (
      editingTestCaseChangeId.value &&
      editingTestCaseChangeId.value !== editingTestCase.value.change_id
    ) {
      mutation = await updateTestCaseChange(
        editingTestCase.value.id,
        editingTestCaseChangeId.value,
      );
    }
    applyTestCaseMutation(mutation.test_cases, mutation.change);
    testCaseEditOpen.value = false;
    editingTestCase.value = null;
    editingTestCaseScenario.value = '';
    editingTestCaseChangeId.value = null;
    await changeCache.loadProjectChanges(mutation.change.project_id);
    if (mutation.change.id !== changeId.value) await loadChangeDetail();
  } catch (err) {
    testCaseError.value = err instanceof Error ? err.message : 'Unable to update test case.';
  } finally {
    savingTestCase.value = false;
  }
}

function openChangeEdit(changeID: number) {
  void router.push(`/changes/edit/${changeID}`);
}

function formatModified(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function completionLabel(change: Change) {
  return `${change.done_tc}/${change.total_tc} - ${change.completed}%`;
}

function epicName(change: Change) {
  if (!change.epic_id) return 'Standalone';
  return (
    change.epic_name ||
    epics.value.find((epic) => epic.id === change.epic_id)?.name ||
    `#${change.epic_id}`
  );
}

onMounted(() => {
  void ensureProjectsLoaded();
  void loadChangeDetail();
});

watch(changeId, () => {
  closeTestCaseEdit();
  closeTestCaseCreate();
  projectMismatchOpen.value = false;
  projectMismatch.value = null;
  void ensureProjectsLoaded();
  void loadChangeDetail();
});
</script>

<style scoped lang="scss">
.test-case-edit-dialog {
  width: min(720px, calc(100vw - 32px));
}

.test-case-edit-fields {
  display: grid;
  gap: 16px;
}

.change-detail-table {
  margin-top: 16px;
}

.change-actions-cell {
  width: 64px;
}

.change-detail-artifact-cell {
  min-height: 160px;
}

.change-detail-text {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
}
</style>
