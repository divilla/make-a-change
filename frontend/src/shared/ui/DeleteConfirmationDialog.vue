<template>
  <q-dialog :model-value="open" persistent @update:model-value="$emit('update:open', Boolean($event))">
    <q-card class="delete-confirmation-dialog">
      <q-card-section>
        <div class="text-h6">Are you sure?</div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Cancel" v-close-popup />
        <q-btn label="OK" color="negative" @click="$emit('confirm')" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean;
}>();

defineEmits<{
  'update:open': [value: boolean];
  confirm: [];
}>();
</script>

<style scoped lang="scss">
.delete-confirmation-dialog {
  width: min(360px, calc(100vw - 32px));
}
</style>
