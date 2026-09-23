<template>
  <div class="epics-toolbar">
    <q-btn color="primary" icon="add" label="New Epic" no-caps @click="$emit('create')" />
  </div>

  <q-markup-table flat bordered class="epic-list-table">
    <thead>
      <tr>
        <th class="text-right">nr</th>
        <th class="text-left">Epic</th>
        <th class="text-center">Complete</th>
        <th class="text-center">Changes</th>
        <th class="text-center">Version</th>
        <th class="text-left">Created</th>
        <th class="text-left">Modified</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="epic in epics" :key="epic.id">
        <td class="text-right">#{{ epic.id }}</td>
        <td class="text-left">{{ epic.name }}</td>
        <td class="text-center">{{ epic.done_tc }}/{{ epic.total_tc }} - {{ epic.completed }}%</td>
        <td class="text-center">{{ epic.change_count }}</td>
        <td class="text-center">{{ epic.version }}</td>
        <td class="text-left">{{ formatEpicDate(epic.created) }}</td>
        <td class="text-left">{{ formatEpicDate(epic.modified) }}</td>
        <td class="text-right">
          <q-btn dense flat round icon="edit" aria-label="Edit epic" @click="$emit('edit', epic)">
            <q-tooltip>Edit epic</q-tooltip>
          </q-btn>
          <q-btn
            dense
            flat
            round
            icon="delete"
            color="negative"
            aria-label="Delete epic"
            :disable="epic.change_count > 0"
            @click="$emit('delete', epic)"
          >
            <q-tooltip>
              {{ epic.change_count > 0 ? 'Delete all linked changes before deleting this epic' : 'Delete epic' }}
            </q-tooltip>
          </q-btn>
        </td>
      </tr>
    </tbody>
  </q-markup-table>

  <div v-if="!epics.length && !loading" class="empty-state compact-empty">
    <q-icon name="view_timeline" size="32px" />
  </div>
</template>

<script setup lang="ts">
import type { Epic } from '../model/epic.types';

function formatEpicDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

defineProps<{
  epics: Epic[];
  loading: boolean;
}>();

defineEmits<{
  create: [];
  edit: [epic: Epic];
  delete: [epic: Epic];
}>();
</script>
