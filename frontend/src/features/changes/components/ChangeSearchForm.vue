<template>
  <form v-if="visible" class="change-search-row" @submit.prevent="$emit('search')">
    <q-btn
      color="primary"
      icon="add_circle"
      type="button"
      :loading="loading"
      no-caps
      label="New Change"
      @click="$emit('new-change')"
    />
    <q-input
      :model-value="title"
      dense
      outlined
      label="Change title"
      class="create-input"
      @update:model-value="(value) => $emit('update:title', value == null ? '' : String(value))"
    />
    <q-select
      :model-value="changeType"
      dense
      outlined
      emit-value
      map-options
      clearable
      label="Type"
      :options="typeOptions"
      class="change-select"
      @update:model-value="(value) => $emit('update:changeType', value == null ? '' : String(value))"
    />
    <q-select
      :model-value="changePhase"
      dense
      outlined
      emit-value
      map-options
      clearable
      label="Phase"
      :options="phaseOptions"
      class="change-select"
      @update:model-value="(value) => $emit('update:changePhase', value == null ? '' : String(value))"
    />
    <q-btn color="primary" icon="search" type="submit" :loading="loading" no-caps label="Search" />
    <q-btn
      color="negative"
      icon="clear"
      type="button"
      :loading="loading"
      outline
      no-caps
      label="Clear"
      @click="$emit('clear')"
    />
  </form>
</template>

<script setup lang="ts">
import type { SelectOption } from '../model/change.types';

defineProps<{
  visible: boolean;
  title: string;
  changeType: string;
  changePhase: string;
  typeOptions: SelectOption[];
  phaseOptions: SelectOption[];
  loading: boolean;
}>();

defineEmits<{
  'update:title': [value: string];
  'update:changeType': [value: string];
  'update:changePhase': [value: string];
  'new-change': [];
  search: [];
  clear: [];
}>();
</script>
