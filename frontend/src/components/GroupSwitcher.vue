<template>
  <div class="group-switcher" ref="root">
    <button
      type="button"
      class="group-switcher-trigger"
      :class="{ 'group-switcher-trigger--active': isOpen }"
      @click="toggleOpen"
      aria-haspopup="true"
      :aria-expanded="isOpen"
      aria-label="Switch group"
    >
      <span class="group-switcher-name">{{ activeGroupName }}</span>
      <svg class="group-switcher-chevron" :class="{ 'group-switcher-chevron--open': isOpen }"
        viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    <transition name="switcher-panel">
      <div v-if="isOpen" class="group-switcher-panel" role="menu">
        <ul v-if="groups.length" class="switcher-group-list">
          <li v-for="group in groups" :key="group.id">
            <button
              type="button"
              class="switcher-group-item"
              :class="{ 'switcher-group-item--active': group.id === activeGroupId }"
              role="menuitem"
              @click="selectGroup(group.id)"
            >
              <span class="switcher-group-item-name">{{ group.name }}</span>
              <svg v-if="group.id === activeGroupId" class="switcher-check-icon" viewBox="0 0 24 24"
                fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 12l2 2 4-4" />
                <circle cx="12" cy="12" r="10" />
              </svg>
            </button>
          </li>
        </ul>
        <p v-else class="switcher-empty-text">You're not in a group yet.</p>

        <div class="switcher-separator"></div>

        <form class="switcher-join-form" @submit.prevent="submitJoin">
          <input
            v-model="joinCode"
            class="form-input switcher-join-input"
            type="text"
            placeholder="Invite code"
            autocapitalize="characters"
            aria-label="Invite code"
            :disabled="isJoining"
          >
          <button
            class="btn-base btn-primary btn-small switcher-join-btn"
            type="submit"
            :disabled="isJoining || !joinCode.trim()"
          >
            {{ isJoining ? 'Joining...' : 'Join' }}
          </button>
        </form>
        <p v-if="joinError" class="error-message switcher-error">{{ joinError }}</p>

        <button type="button" class="switcher-create-btn" @click="openCreateModal">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          Create a group
        </button>
      </div>
    </transition>

    <CreateGroupModal v-if="showCreateModal" @close="showCreateModal = false" />
  </div>
</template>

<script>
import { joinGroup } from '@/services/api';
import { setActiveGroupId } from '@/services/activeGroup';
import CreateGroupModal from '@/components/CreateGroupModal.vue';

export default {
  name: 'GroupSwitcher',
  components: { CreateGroupModal },
  props: {
    groups: { type: Array, default: () => [] },
    activeGroupId: { type: String, default: '' }
  },
  data() {
    return {
      isOpen: false,
      joinCode: '',
      isJoining: false,
      joinError: '',
      showCreateModal: false
    };
  },
  computed: {
    activeGroupName() {
      if (!this.groups.length) {
        return 'No group yet';
      }
      const active = this.groups.find((group) => group.id === this.activeGroupId);
      return active ? active.name : 'Select a group';
    }
  },
  mounted() {
    document.addEventListener('click', this.handleClickOutside);
    document.addEventListener('keydown', this.handleKeydown);
  },
  beforeUnmount() {
    document.removeEventListener('click', this.handleClickOutside);
    document.removeEventListener('keydown', this.handleKeydown);
  },
  methods: {
    toggleOpen() {
      this.isOpen = !this.isOpen;
    },
    handleClickOutside(event) {
      if (this.isOpen && this.$refs.root && !this.$refs.root.contains(event.target)) {
        this.isOpen = false;
      }
    },
    handleKeydown(event) {
      if (this.isOpen && event.key === 'Escape') {
        this.isOpen = false;
      }
    },
    // Same behaviour as the old <select>'s @change: there is no reactive
    // store, every scoped view reads the active group once in created(), so
    // a full reload is what actually re-scopes the app.
    selectGroup(groupId) {
      this.isOpen = false;
      if (!groupId || groupId === this.activeGroupId) {
        return;
      }
      setActiveGroupId(groupId);
      window.location.reload();
    },
    async submitJoin() {
      const code = this.joinCode.trim();
      if (!code) {
        return;
      }
      this.isJoining = true;
      this.joinError = '';
      try {
        const group = await joinGroup(code);
        // A much better default than joining and staying scoped to whatever
        // was active before — switch straight into the group just joined.
        setActiveGroupId(group.id);
        window.location.reload();
      } catch (error) {
        this.joinError = this.backendMessage(error, 'Failed to join the group.');
        this.isJoining = false;
      }
    },
    openCreateModal() {
      this.isOpen = false;
      this.showCreateModal = true;
    },
    backendMessage(error, fallback) {
      return error.response?.data?.error || fallback;
    }
  }
};
</script>

<style scoped>
.group-switcher {
  position: relative;
}

.group-switcher-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  max-width: 12rem;
  transition: all var(--transition-fast);
}

.group-switcher-trigger:hover,
.group-switcher-trigger--active {
  border-color: var(--primary-color);
}

.group-switcher-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-switcher-chevron {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--text-secondary);
  transition: transform var(--transition-fast);
}

.group-switcher-chevron--open {
  transform: rotate(180deg);
}

.group-switcher-panel {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  width: 17rem;
  max-width: calc(100vw - 2rem);
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-xl);
  padding: 0.5rem;
  z-index: 1001;
}

.switcher-group-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 16rem;
  overflow-y: auto;
}

.switcher-group-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: none;
  border: none;
  border-radius: var(--border-radius);
  color: var(--text-primary);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.switcher-group-item:hover {
  background-color: var(--bg-tertiary);
}

.switcher-group-item--active {
  color: var(--primary-color);
}

.switcher-group-item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.switcher-check-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--primary-color);
}

.switcher-empty-text {
  color: var(--text-secondary);
  font-size: 0.875rem;
  padding: 0.5rem 0.75rem;
  margin: 0;
}

.switcher-separator {
  height: 1px;
  background-color: var(--border-color);
  margin: 0.5rem 0.25rem;
}

.switcher-join-form {
  display: flex;
  gap: 0.5rem;
  padding: 0.25rem 0.25rem 0;
}

.switcher-join-input {
  flex: 1;
  padding: 0.5rem 0.6rem;
  font-size: 0.875rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.switcher-join-btn {
  flex-shrink: 0;
  white-space: nowrap;
}

.switcher-error {
  margin: 0.5rem 0.25rem 0;
  font-size: 0.8rem;
}

.switcher-create-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  margin-top: 0.5rem;
  padding: 0.6rem 0.75rem;
  background: none;
  border: none;
  border-radius: var(--border-radius);
  color: var(--primary-color);
  font-size: 0.9rem;
  font-weight: 600;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.switcher-create-btn:hover {
  background-color: var(--bg-tertiary);
}

.switcher-create-btn svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.switcher-panel-enter-active,
.switcher-panel-leave-active {
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.switcher-panel-enter-from,
.switcher-panel-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

@media (max-width: 768px) {
  .group-switcher-trigger {
    max-width: 8rem;
    padding: 0.5rem;
  }

  .group-switcher-panel {
    right: -0.5rem;
    width: 15rem;
  }
}
</style>
