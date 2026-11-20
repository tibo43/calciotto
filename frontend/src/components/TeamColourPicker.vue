<template>
  <!-- A single swatch showing the current colour (or a rainbow placeholder
       when unset), with the browser's own native colour picker stretched
       invisibly over it — no preset list, just "pick whatever colour you
       want" via the OS picker. -->
  <label class="colour-swatch" :class="{ 'colour-swatch--disabled': disabled }"
    :style="modelValue ? { background: modelValue } : {}" title="Pick a colour">
    <span class="colour-swatch-icon" :style="{ color: iconColour }">&#9998;</span>
    <input type="color" class="colour-swatch-native-input" :value="modelValue || '#000000'" :disabled="disabled"
      @input="$emit('update:modelValue', $event.target.value)">
  </label>
</template>

<script>
function isLightColour(hex) {
  if (!hex || hex.length !== 7) {
    return false;
  }
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return (0.299 * r + 0.587 * g + 0.114 * b) / 255 > 0.6;
}

export default {
  name: 'TeamColourPicker',
  props: {
    modelValue: { type: String, default: '' },
    disabled: { type: Boolean, default: false }
  },
  emits: ['update:modelValue'],
  computed: {
    // The pencil icon needs to stay legible over whatever colour is picked
    // (and over the rainbow placeholder, where it reads as dark).
    iconColour() {
      return this.modelValue && isLightColour(this.modelValue) ? '#1f2937' : '#ffffff';
    }
  }
};
</script>

<style scoped>
.colour-swatch {
  position: relative;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  padding: 0;
  border-radius: 50%;
  border: 2px solid var(--border-color);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: conic-gradient(from 180deg, #ef4444, #f59e0b, #10b981, #06b6d4, #3b82f6, #8b5cf6, #ec4899, #ef4444);
  transition: transform var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.colour-swatch:hover {
  transform: scale(1.08);
  border-color: var(--primary-color);
}

.colour-swatch--disabled {
  cursor: not-allowed;
  opacity: 0.55;
  pointer-events: none;
}

.colour-swatch-icon {
  font-size: 1rem;
  line-height: 1;
  pointer-events: none;
  text-shadow: 0 0 3px rgba(0, 0, 0, 0.35);
}

.colour-swatch-native-input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  padding: 0;
  border: 0;
  opacity: 0;
  cursor: pointer;
}

.colour-swatch-native-input:disabled {
  cursor: not-allowed;
}
</style>
