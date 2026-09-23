<template>
  <form class="create-row" @submit.prevent="$emit('create')">
    <q-input
      :model-value="modelValue"
      dense
      outlined
      label="Project name"
      class="create-input"
      @update:model-value="(value) => $emit('update:modelValue', value == null ? '' : String(value))"
    />
    <q-btn
      color="primary"
      icon="add"
      type="submit"
      :disable="!modelValue.trim()"
      round
      unelevated
    >
      <q-tooltip>Create project</q-tooltip>
    </q-btn>
  </form>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string;
}>();

defineEmits<{
  'update:modelValue': [value: string];
  create: [];
}>();
</script>
