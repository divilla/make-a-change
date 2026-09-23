<template>
  <q-dialog :model-value="open" @update:model-value="$emit('update:open', Boolean($event))">
    <q-card class="dialog-card">
      <q-card-section>
        <div class="text-subtitle1">Rename Project</div>
      </q-card-section>
      <q-card-section>
        <q-input
          :model-value="name"
          autofocus
          outlined
          label="Project name"
          @update:model-value="(value) => $emit('update:name', value == null ? '' : String(value))"
        />
      </q-card-section>
      <q-card-actions align="right">
        <q-btn flat label="Cancel" no-caps v-close-popup />
        <q-btn
          color="primary"
          label="Save"
          no-caps
          :disable="!name.trim()"
          @click="$emit('save')"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean;
  name: string;
}>();

defineEmits<{
  'update:open': [value: boolean];
  'update:name': [value: string];
  save: [];
}>();
</script>
