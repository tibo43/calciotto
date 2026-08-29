<template>
  <div class="colour-picker">
    <button v-for="hex in presets" :key="hex" type="button" class="colour-swatch"
      :class="{ 'colour-swatch--selected': hex === modelValue }" :style="{ backgroundColor: hex }"
      :disabled="disabled" :aria-pressed="hex === modelValue" :aria-label="'Use colour ' + hex"
      @click="$emit('update:modelValue', hex)">
      <span v-if="hex === modelValue" class="colour-swatch-check" :style="{ color: checkColour(hex) }">&#10003;</span>
    </button>

    <!-- Escape hatch: any colour outside the 10 presets. Styled as a
         rainbow swatch when inactive, or as a plain swatch of the current
         custom value (with the same selected ring/check as a preset) when
         the bound value isn't one of the presets. The native colour input
         is stretched invisibly over the label so clicking it opens the
         browser's picker, same trigger the old <input type="color"> gave. -->
    <label class="colour-swatch colour-swatch--custom"
      :class="{ 'colour-swatch--selected': isCustom, 'colour-swatch--disabled': disabled }"
      :style="isCustom ? { background: modelValue } : {}" title="Pick a custom colour">
      <span v-if="isCustom" class="colour-swatch-check" :style="{ color: checkColour(modelValue) }">&#10003;</span>
      <span v-else class="colour-swatch-plus">+</span>
      <input type="color" class="colour-swatch-native-input" :value="modelValue || '#000000'" :disabled="disabled"
        @input="$emit('update:modelValue', $event.target.value)">
    </label>
  </div>
</template>

<script>
// Exactly the 10 legacy keyword colours' hex values, in the same order
// getTeamColor() (MatchesAll.vue/MatchDetails.vue) maps them from —
// picking a preset here must produce the exact same hex a legacy
// keyword-coloured team already renders as.
const PRESET_COLOURS = [
  '#ef4444', '#3b82f6', '#10b981', '#f59e0b', '#8b5cf6',
  '#f97316', '#ec4899', '#06b6d4', '#f8fafc', '#1f2937'
];

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
  data() {
    return { presets: PRESET_COLOURS };
  },
  computed: {
    isCustom() {
      return Boolean(this.modelValue) && !this.presets.includes(this.modelValue);
    }
  },
  methods: {
    checkColour(hex) {
      return isLightColour(hex) ? '#1f2937' : '#ffffff';
    }
  }
};
</script>

<style scoped>
.colour-picker {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.colour-swatch {
  position: relative;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  padding: 0;
  border-radius: 50%;
  border: 2px solid var(--border-color);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--transition-fast), border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.colour-swatch:hover {
  transform: scale(1.1);
  border-color: var(--primary-color);
}

.colour-swatch--selected {
  border-width: 3px;
  border-color: var(--primary-color);
  box-shadow: var(--shadow-sm);
}

.colour-swatch:disabled,
.colour-swatch--disabled {
  cursor: not-allowed;
  opacity: 0.55;
  pointer-events: none;
}

.colour-swatch-check {
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1;
  pointer-events: none;
}

.colour-swatch--custom {
  background: conic-gradient(from 180deg, #ef4444, #f59e0b, #10b981, #06b6d4, #3b82f6, #8b5cf6, #ec4899, #ef4444);
  overflow: hidden;
}

.colour-swatch-plus {
  color: #ffffff;
  font-weight: 700;
  font-size: 1rem;
  line-height: 1;
  text-shadow: 0 0 3px rgba(0, 0, 0, 0.55);
  pointer-events: none;
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
