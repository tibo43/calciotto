<template>
  <main class="page">
    <section v-if="loading" class="state">
      <div class="skel md"></div>
      <div class="skel lg"></div>
    </section>
    <section v-else-if="failed" class="state">
      <p class="label">Error</p>
      <h1>Couldn't load your profile</h1>
      <p>Check your connection, then try again.</p>
      <button class="btn" type="button" @click="load">Try again</button>
    </section>

    <template v-else-if="page === 'home' || page === 'group'">
      <section v-if="page === 'home'" class="ident">
        <div class="avatar" aria-hidden="true">{{ initials }}</div>
        <div>
          <p class="label">Account</p>
          <h1>{{ displayName(record.Name) }}</h1>
          <router-link class="account-link" to="/profile/account">Manage account</router-link>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Record</h2></div>
        <div class="season-field">
          <label class="sr-only" for="profile-season">Season</label>
          <select id="profile-season" aria-label="Season" :value="season" @change="onSeason">
            <option value="">All seasons</option>
            <option v-for="item in seasonOptions" :key="item" :value="item">{{ item }}</option>
          </select>
        </div>
        <p class="scope">{{ scopeText }}</p>
        <div class="stats">
          <div v-for="stat in stats" :key="stat.label" class="stat">
            <div class="n">{{ stat.value }}</div>
            <p class="label l">{{ stat.label }}</p>
          </div>
        </div>
      </section>
      <section v-if="page === 'home'" class="sec">
        <div class="sec-head"><h2>Groups</h2></div>
        <p v-if="rows.length === 0" class="scope">No group yet. Ask an admin to invite you.</p>
        <div v-for="row in rows" :key="row.GroupID" class="group-row">
          <button class="fav-star" :class="{ 'is-off': !isFavorite(row) }" type="button"
            :aria-label="isFavorite(row) ? 'Favourite group' : 'Set as favourite'"
            @click="favourite(row)">
            <svg v-if="isFavorite(row)" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.6"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>
          </button>
          <router-link class="grow" :to="`/profile/groups/${row.GroupID}`">
            <h3>{{ row.GroupName }}
              <span v-if="isFavorite(row)" class="tag tag-acc">Favourite</span>
              <span class="tag">{{ roleOf(row) === 'admin' ? 'Admin' : 'Member' }}</span>
            </h3>
            <p class="meta">{{ row.Played }} matches · {{ row.Points }} pts</p>
          </router-link>
          <router-link v-if="roleOf(row) === 'admin'" class="group-gear" :to="`/profile/groups/${row.GroupID}/settings`" aria-label="Group settings">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
          </router-link>
          <router-link class="chev-link" :to="`/profile/groups/${row.GroupID}`" aria-hidden="true" tabindex="-1">
            <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 2l4 4-4 4"/></svg>
          </router-link>
        </div>
      </section>
      <section v-else-if="isAdmin" class="sec">
        <router-link class="btn btn-ghost" :to="`/profile/groups/${groupId}/settings`">Group settings</router-link>
      </section>
      <section v-if="page === 'home'" class="sec">
        <div class="sec-head"><h2>Settings</h2></div>
        <div class="pref-row">
          <div>
            <div class="who">Dark mode</div>
            <div class="note">Applies across the site, on this device</div>
          </div>
          <button class="switch" :class="{ 'is-on': dark }" type="button" role="switch" :aria-checked="dark.toString()" aria-label="Dark mode" @click="toggleTheme"><span class="track"></span></button>
        </div>
        <router-link class="pref-row" to="/profile/notifications">
          <div>
            <div class="who">Notifications</div>
            <div class="note">{{ pushOn ? 'On on this device' : 'Off on this device' }}</div>
          </div>
          <span class="switch" :class="{ 'is-on': pushOn }" aria-hidden="true"><span class="track"></span></span>
        </router-link>
        <router-link class="pref-row" to="/profile/account">
          <div>
            <div class="who">Account</div>
            <div class="note">Name, email, password</div>
          </div>
          <svg class="chev" width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 2l4 4-4 4"/></svg>
        </router-link>
      </section>
      <div v-if="page === 'home'" class="logout-row">
        <button class="btn btn-ghost btn-block" type="button" @click="logout">Log out</button>
      </div>
    </template>

    <template v-else-if="page === 'notifications'">
      <section class="sec">
        <div class="pref-row">
          <div>
            <div class="who">Notifications on this device</div>
            <div class="note">{{ pushNote }}</div>
          </div>
          <button class="switch" :class="{ 'is-on': pushOn }" type="button" role="switch" :aria-checked="pushOn.toString()" aria-label="Notifications on this device" @click="togglePush"><span class="track"></span></button>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Types</h2></div>
        <section v-for="group in preferenceGroups" :key="group.label" class="pref-group" :class="{ 'is-locked': !pushOn }">
          <p class="label">{{ group.label }}</p>
          <div v-for="item in group.items" :key="item.key" class="pref-row">
            <div class="who">{{ item.label }}</div>
            <button class="switch" :class="{ 'is-on': prefOn(item.key) }" type="button" role="switch" :disabled="!pushOn" :aria-checked="prefOn(item.key).toString()" :aria-label="item.label" @click="togglePref(item.key)"><span class="track"></span></button>
          </div>
        </section>
      </section>
    </template>

    <template v-else-if="page === 'account'">
      <section class="sec">
        <div class="sec-head"><h2>Identity</h2></div>
        <div class="field">
          <label class="label" for="account-name">Name</label>
          <input id="account-name" v-model="nameInput" type="text" autocomplete="name">
        </div>
        <p><button class="btn" type="button" :disabled="savingName || !nameInput.trim()" @click="saveName">{{ savingName ? 'Saving…' : 'Save name' }}</button></p>
        <p v-if="nameError" class="scope">{{ nameError }}</p>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Email</h2></div>
        <div class="field">
          <label class="label" for="account-email">Email</label>
          <input id="account-email" type="email" value="" readonly>
          <p class="hint">Changing your email isn't available yet.</p>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Password</h2></div>
        <p class="scope">To choose a new one, we email you a reset link, the same path as forgot password.</p>
        <p><button class="btn btn-ghost" type="button" @click="sendReset">Email the link</button></p>
        <p v-if="resetNote" class="scope">{{ resetNote }}</p>
      </section>
      <section class="danger">
        <p class="label">Danger zone</p>
        <h2 class="h2">Delete my account</h2>
        <p>This can't be undone. Your name, email and password are erased, and you're removed from every group. Match history (goals, Man of the Match titles) stays for the other players, and is no longer tied to an account.</p>
        <span class="withdraw-wrap">
          <button class="btn btn-danger" type="button" :aria-expanded="deleting.toString()" @click="openDelete">Delete my account</button>
          <div v-if="deleting" class="confirm-pop" role="dialog" aria-labelledby="delete-title">
            <h3 id="delete-title">Confirm deletion</h3>
            <p class="ref">Enter your current password to confirm.</p>
            <div class="field">
              <label class="label" for="delete-password-pop">Password</label>
              <input id="delete-password-pop" v-model="deletePassword" type="password" autocomplete="current-password">
            </div>
            <p v-if="deleteError" class="scope">{{ deleteError }}</p>
            <div class="confirm-actions actions">
              <button class="btn btn-ghost" type="button" @click="closeDelete">Cancel</button>
              <button class="btn btn-danger" type="button" @click="confirmDelete">Delete my account</button>
            </div>
          </div>
        </span>
      </section>
      <div v-if="deleting" class="confirm-scrim" @click="closeDelete"></div>
      <div v-if="deleting" class="confirm-sheet" role="dialog" aria-labelledby="delete-sheet-title">
        <h3 id="delete-sheet-title">Confirm deletion</h3>
        <p class="ref">Enter your current password to confirm.</p>
        <div class="field">
          <label class="label" for="delete-password">Password</label>
          <input id="delete-password" v-model="deletePassword" type="password" autocomplete="current-password">
        </div>
        <p v-if="deleteError" class="scope">{{ deleteError }}</p>
        <div class="confirm-actions">
          <button class="btn btn-danger" type="button" @click="confirmDelete">Delete my account</button>
          <button class="btn btn-ghost" type="button" @click="closeDelete">Cancel</button>
        </div>
      </div>
    </template>

    <template v-else-if="page === 'settings'">
      <section class="sec">
        <div class="sec-head"><h2>Teams</h2></div>
        <p class="scope">Name and colour. Save each team on its own.</p>
        <div v-for="team in teams" :key="team.id" class="team-edit">
          <TeamColourPicker v-model="team.colour" />
          <input v-model="team.name" type="text" :aria-label="`Name of ${team.name}`">
          <button class="btn" type="button" @click="saveTeam(team)">Save</button>
        </div>
        <p v-if="teamNote" class="scope">{{ teamNote }}</p>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Invitation</h2></div>
        <div class="invite-box">
          <code>{{ inviteCode || '—' }}</code>
          <button class="btn btn-ghost" type="button" @click="copyCode">Copy</button>
          <button class="btn btn-ghost" type="button" @click="shareInvite">Invite on WhatsApp</button>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Members</h2></div>
        <p class="scope">You can't change your own role. The last admin can't be demoted.</p>
        <div v-for="member in members" :key="member.id" class="member-row">
          <div>
            <div class="nm" :class="{ 'is-self': member.id === playerId }">
              {{ displayName(member.name) }}
              <span v-if="member.role === 'admin'" class="tag">Admin</span>
              <span v-if="member.id === playerId" class="tag">You</span>
            </div>
          </div>
          <div v-if="member.id !== playerId" class="member-actions withdraw-wrap">
            <button v-if="member.role === 'admin'" class="btn btn-ghost" type="button" :aria-expanded="(demoting === member.id).toString()" @click="askDemote(member)">Remove admin</button>
            <button v-else-if="member.email" class="btn btn-ghost" type="button" @click="promote(member)">Make admin</button>
            <div v-if="demoting === member.id" class="confirm-pop" role="dialog">
              <h3>Remove the admin role?</h3>
              <p class="ref">{{ displayName(member.name) }} stays a member of the group. You can make them an admin again later.</p>
              <div class="confirm-actions actions">
                <button class="btn btn-ghost" type="button" @click="demoting = ''">Cancel</button>
                <button class="btn btn-danger" type="button" @click="confirmDemote(member)">Remove admin</button>
              </div>
            </div>
          </div>
        </div>
        <p v-if="roleError" class="scope">{{ roleError }}</p>
      </section>
      <div v-if="demoting" class="confirm-scrim" @click="demoting = ''"></div>
      <div v-if="demoting" class="confirm-sheet" role="dialog">
        <h3>Remove the admin role?</h3>
        <p class="ref">{{ displayName(demoteName) }} stays a member of the group. You can make them an admin again later.</p>
        <div class="confirm-actions">
          <button class="btn btn-danger" type="button" @click="confirmDemote(demoteMember)">Remove admin</button>
          <button class="btn btn-ghost" type="button" @click="demoting = ''">Cancel</button>
        </div>
      </div>
    </template>
  </main>
</template>

<script>
import {
  getPlayerProfile, getMyGroups, getSeasons, setFavoriteGroup, updateMyName,
  deleteMyAccount, clearToken, getToken, getTeamsByGroup, updateTeam, getInviteCode,
  getGroupMembers, updateMemberRole, getNotificationPreferences, updateNotificationPreferences,
} from '@/services/api';
import { getPushState, enablePush, disablePush, PUSH_STATE } from '@/services/pushNotifications';
import { buildGroupInviteShareText, buildWhatsAppShareUrl } from '@/services/whatsappShare';
import TeamColourPicker from '@/components/TeamColourPicker.vue';

const COLOURS = {
  red: '#ef4444', blue: '#3b82f6', green: '#10b981', yellow: '#f59e0b',
  purple: '#8b5cf6', orange: '#f97316', pink: '#ec4899', cyan: '#06b6d4',
  white: '#f8fafc', black: '#1f2937',
};

const PREFERENCE_GROUPS = [
  { label: 'Match', items: [
    { key: 'registration_open', label: 'Sign-ups open' },
    { key: 'team_assigned', label: 'Your team is set' },
    { key: 'result_recorded', label: 'Final score' },
  ] },
  { label: 'Predictions', items: [
    { key: 'predictions_open', label: 'Predictions open' },
    { key: 'predictions_settled', label: 'Predictions settled' },
  ] },
  { label: 'Reminder', items: [
    { key: 'kickoff_reminder', label: 'Kick-off in 30 minutes' },
  ] },
];

function playerIdFromToken() {
  const token = getToken();
  if (!token) return '';
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch (error) {
    return '';
  }
}

function toHex(colour) {
  if (colour && String(colour).startsWith('#')) return colour;
  return COLOURS[String(colour || '').toLowerCase()] || '#6b7280';
}

export default {
  name: 'ProfileV2',
  components: { TeamColourPicker },
  inject: {
    v2Context: { default: () => ({}) },
  },
  data() {
    return {
      loading: true,
      failed: false,
      overall: {},
      perGroup: [],
      seasons: [],
      season: '',
      groups: [],
      playerId: '',
      pushState: PUSH_STATE.OFF,
      preferences: {},
      preferenceGroups: PREFERENCE_GROUPS,
      nameInput: '',
      savingName: false,
      nameError: '',
      resetNote: '',
      deleting: false,
      deletePassword: '',
      deleteError: '',
      teams: [],
      inviteCode: '',
      members: [],
      teamNote: '',
      demoting: '',
      roleError: '',
    };
  },
  computed: {
    page() {
      const name = this.$route.name;
      if (name === 'ProfileGroup') return 'group';
      if (name === 'ProfileGroupSettings') return 'settings';
      if (name === 'ProfileNotifications') return 'notifications';
      if (name === 'ProfileAccount') return 'account';
      return 'home';
    },
    groupId() {
      return this.$route.params.id || '';
    },
    record() {
      if (this.page === 'group') {
        return this.perGroup.find((row) => row.GroupID === this.groupId) || {};
      }
      return this.overall || {};
    },
    rows() {
      return this.perGroup;
    },
    seasonOptions() {
      return [...this.seasons].reverse();
    },
    scopeText() {
      if (this.page === 'group') return this.record.GroupName || 'This group';
      if (this.rows.length === 1) return this.rows[0].GroupName;
      return `${this.rows.length} groups combined`;
    },
    isAdmin() {
      return this.roleOf(this.record) === 'admin';
    },
    initials() {
      return String(this.record.Name || '')
        .split(' ').filter(Boolean).slice(0, 2).map((part) => part.charAt(0).toUpperCase()).join('') || '?';
    },
    stats() {
      const row = this.record;
      return [
        { label: 'Wins', value: row.Won || 0 },
        { label: 'Draws', value: row.Drawn || 0 },
        { label: 'Losses', value: row.Lost || 0 },
        { label: 'Points', value: row.Points || 0 },
        { label: 'Goals', value: row.GoalsFor || 0 },
        { label: 'OG', value: row.OwnGoals || 0 },
        { label: 'MOTM', value: row.MotmAwards || 0 },
        { label: 'Pred. pts', value: this.predictionLabel(row.PredictionPoints) },
      ];
    },
    dark() {
      return Boolean(this.$root.isDarkMode);
    },
    pushOn() {
      return this.pushState === PUSH_STATE.ON;
    },
    pushNote() {
      if (this.pushState === PUSH_STATE.NEEDS_INSTALL) return 'Install the app first, then turn notifications on.';
      if (this.pushState === PUSH_STATE.DENIED) return 'Notifications are blocked in the browser settings.';
      if (this.pushState === PUSH_STATE.UNSUPPORTED) return 'This browser cannot receive notifications.';
      return 'Choose which ones you want.';
    },
    demoteMember() {
      return this.members.find((member) => member.id === this.demoting) || {};
    },
    demoteName() {
      return this.demoteMember.name || '';
    },
  },
  watch: {
    '$route.fullPath'() {
      this.syncChrome();
      if (this.page === 'settings') this.loadSettings();
      if (this.page === 'notifications') this.loadNotifications();
    },
    demoting(value) {
      if (this.v2Context) this.v2Context.confirming = Boolean(value) || this.deleting;
    },
    deleting(value) {
      if (this.v2Context) this.v2Context.confirming = Boolean(value) || Boolean(this.demoting);
    },
  },
  created() {
    this.playerId = playerIdFromToken();
    this.load();
  },
  beforeUnmount() {
    if (!this.v2Context) return;
    this.v2Context.hide = false;
    this.v2Context.pageTitle = '';
    this.v2Context.backTo = '';
    this.v2Context.confirming = false;
  },
  methods: {
    displayName(name) {
      return String(name || '').split(' ').filter(Boolean)
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    },
    predictionLabel(points) {
      const rounded = Math.round(Number(points || 0) * 10) / 10;
      const text = Math.abs(rounded).toFixed(1);
      if (rounded > 0) return `+${text}`;
      if (rounded < 0) return `−${text}`;
      return '0.0';
    },
    roleOf(row) {
      const group = this.groups.find((item) => item.id === (row.GroupID || row.id));
      return group ? group.role : 'member';
    },
    isFavorite(row) {
      const group = this.groups.find((item) => item.id === row.GroupID);
      return Boolean(group && group.is_favorite);
    },
    syncChrome() {
      if (!this.v2Context) return;
      this.v2Context.hide = true;
      const titles = {
        group: this.record.GroupName || 'Group',
        settings: 'Group settings',
        notifications: 'Notifications',
        account: 'Account',
      };
      this.v2Context.pageTitle = titles[this.page] || '';
      this.v2Context.backTo = this.page === 'settings'
        ? `/profile/groups/${this.groupId}`
        : '/profile';
    },
    async load() {
      this.loading = true;
      this.failed = false;
      try {
        const [profile, groups] = await Promise.all([
          getPlayerProfile(this.season),
          getMyGroups(),
        ]);
        this.overall = profile.Overall || {};
        this.perGroup = Array.isArray(profile.PerGroup) ? profile.PerGroup : [];
        this.groups = Array.isArray(groups) ? groups : [];
        this.nameInput = this.overall.Name || '';
        if (this.page === 'group') {
          this.seasons = await getSeasons(this.groupId).catch(() => []);
        } else {
          this.seasons = Array.isArray(profile.Seasons) ? profile.Seasons : [];
        }
        this.pushState = await getPushState().catch(() => PUSH_STATE.UNSUPPORTED);
        this.syncChrome();
        if (this.page === 'settings') await this.loadSettings();
        if (this.page === 'notifications') await this.loadNotifications();
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async onSeason(event) {
      this.season = event.target.value;
      this.loading = true;
      try {
        const profile = await getPlayerProfile(this.season);
        this.overall = profile.Overall || {};
        this.perGroup = Array.isArray(profile.PerGroup) ? profile.PerGroup : [];
      } catch (error) {
        console.error(error);
        this.failed = true;
      } finally {
        this.loading = false;
      }
    },
    async favourite(row) {
      if (this.isFavorite(row)) return;
      await setFavoriteGroup(row.GroupID);
      this.groups = await getMyGroups();
    },
    toggleTheme() {
      if (this.$root.toggleTheme) this.$root.toggleTheme();
    },
    logout() {
      if (this.$root.logout) this.$root.logout();
    },
    async loadNotifications() {
      const prefs = await getNotificationPreferences().catch(() => ({}));
      this.preferences = prefs.preferences || prefs || {};
      this.pushState = await getPushState().catch(() => PUSH_STATE.UNSUPPORTED);
    },
    prefOn(key) {
      return this.preferences[key] !== false;
    },
    async togglePref(key) {
      if (!this.pushOn) return;
      const next = { ...this.preferences, [key]: !this.prefOn(key) };
      this.preferences = next;
      try {
        const saved = await updateNotificationPreferences(next);
        this.preferences = (saved && saved.preferences) || next;
      } catch (error) {
        console.error(error);
      }
    },
    async togglePush() {
      this.pushState = this.pushOn ? await disablePush() : await enablePush();
    },
    async saveName() {
      this.savingName = true;
      this.nameError = '';
      try {
        await updateMyName(this.nameInput.trim());
        await this.load();
      } catch (error) {
        this.nameError = (error.response && error.response.data && error.response.data.error) || 'Could not save the name.';
      } finally {
        this.savingName = false;
      }
    },
    sendReset() {
      this.resetNote = 'Changing the password from here needs the account email, and that address is not available on this page yet.';
    },
    openDelete() {
      this.deleting = true;
      this.deleteError = '';
    },
    closeDelete() {
      this.deleting = false;
      this.deletePassword = '';
    },
    async confirmDelete() {
      this.deleteError = '';
      try {
        await deleteMyAccount(this.deletePassword);
        clearToken();
        window.location.href = '/login?accountDeleted=success';
      } catch (error) {
        this.deleteError = (error.response && error.response.data && error.response.data.error) || 'Could not delete the account.';
      }
    },
    async loadSettings() {
      if (this.roleOf({ GroupID: this.groupId }) !== 'admin') {
        this.$router.replace(`/profile/groups/${this.groupId}`);
        return;
      }
      const [teams, code, members] = await Promise.all([
        getTeamsByGroup(this.groupId).catch(() => []),
        getInviteCode(this.groupId).catch(() => ({ invite_code: '' })),
        getGroupMembers(this.groupId).catch(() => []),
      ]);
      this.teams = (Array.isArray(teams) ? teams : []).map((team) => ({
        id: team.id || team.ID,
        name: team.name || team.Name,
        colour: toHex(team.colour || team.Colour),
      }));
      this.inviteCode = (code && (code.invite_code || code.InviteCode)) || '';
      this.members = Array.isArray(members) ? members : [];
      this.syncChrome();
    },
    async saveTeam(team) {
      this.teamNote = '';
      try {
        await updateTeam(this.groupId, team.id, team.name.trim(), team.colour);
        this.teamNote = 'Saved.';
      } catch (error) {
        this.teamNote = 'Could not save the team.';
      }
    },
    async copyCode() {
      try {
        await navigator.clipboard.writeText(this.inviteCode);
        this.teamNote = 'Code copied.';
      } catch (error) {
        this.teamNote = this.inviteCode;
      }
    },
    shareInvite() {
      const inviteUrl = `${window.location.origin}/signup?invite=${this.inviteCode}`;
      const text = buildGroupInviteShareText({ inviteUrl, groupInviteCode: this.inviteCode });
      window.open(buildWhatsAppShareUrl(text), '_blank', 'noopener');
    },
    askDemote(member) {
      this.demoting = member.id;
      this.roleError = '';
    },
    async confirmDemote(member) {
      if (!member || !member.id) return;
      try {
        await updateMemberRole(this.groupId, member.id, 'member');
        this.demoting = '';
        await this.loadSettings();
      } catch (error) {
        this.roleError = (error.response && error.response.data && error.response.data.error) || 'Could not change the role.';
      }
    },
    async promote(member) {
      this.roleError = '';
      try {
        await updateMemberRole(this.groupId, member.id, 'admin');
        await this.loadSettings();
      } catch (error) {
        this.roleError = (error.response && error.response.data && error.response.data.error) || 'Could not change the role.';
      }
    },
  },
};
</script>
