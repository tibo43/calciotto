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
        <div class="ident-name">
          <p class="label">Account</p>
          <h1>{{ displayName(record.Name) }}</h1>
        </div>
        <GearLink class="group-gear ident-gear" to="/profile/account" label="Manage account" />
      </section>
      <section v-if="page === 'group' && (currentGroup?.description || currentGroup?.website_url)" class="sec">
        <div class="sec-head"><h2>About this group</h2></div>
        <p v-if="currentGroup.description" class="group-description">{{ currentGroup.description }}</p>
        <a v-if="currentGroup.website_url" class="group-website" :href="currentGroup.website_url"
          target="_blank" rel="noopener noreferrer">
          <span>{{ currentGroup.website_label || 'Open group link' }}</span>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor"
            stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M4 12l8-8M6 4h6v6" />
          </svg>
        </a>
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
          <GearLink v-if="roleOf(row) === 'admin'" class="group-gear" :to="`/profile/groups/${row.GroupID}/settings`" label="Group settings" />
          <router-link class="chev-link" :to="`/profile/groups/${row.GroupID}`" aria-hidden="true" tabindex="-1">
            <svg class="chev" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 2l4 4-4 4"/></svg>
          </router-link>
        </div>
      </section>
      <section v-if="page === 'home'" class="sec">
        <div class="sec-head"><h2>Settings</h2></div>
        <router-link v-if="hasPush" class="pref-row" to="/profile/notifications">
          <div>
            <div class="who">Notifications</div>
            <div class="note">{{ pushOn ? 'On on this device' : 'Off on this device' }}</div>
          </div>
          <span class="switch" :class="{ 'is-on': pushOn }" aria-hidden="true"><span class="track"></span></span>
        </router-link>
        <!-- Hidden feature pwa-install, like today's install card: shown only
             where the browser can install the app (see pwaInstall.js). -->
        <div v-if="installOffered" class="pref-row">
          <div>
            <div class="who">Install Calciotto</div>
            <div v-if="installState === 'prompt'" class="note">Open it from your home screen, like any other app</div>
            <div v-else class="note">Tap Share, then Add to Home Screen</div>
          </div>
          <button v-if="installState === 'prompt'" class="btn btn-ghost" type="button" @click="install">Install</button>
        </div>
        <button class="pref-row guide-action" type="button" @click="openGuide">
          <span class="who">Replay app guide</span>
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor"
            stroke-width="1.6" aria-hidden="true"><path d="m6 3 5 5-5 5" /></svg>
        </button>
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
          <button class="switch" :class="{ 'is-on': pushOn }" type="button" role="switch" :disabled="!canTogglePush" :aria-checked="pushOn.toString()" aria-label="Notifications on this device" @click="togglePush"><span class="track"></span></button>
        </div>
        <p v-if="pushError" class="scope">{{ pushError }}</p>
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
        <div class="sec-head"><h2>Appearance</h2></div>
        <div class="pref-row">
          <div>
            <div class="who">Dark mode</div>
            <div class="note">Applies across the site, on this device</div>
          </div>
          <button class="switch" :class="{ 'is-on': dark }" type="button" role="switch" :aria-checked="dark.toString()" aria-label="Dark mode" @click="toggleTheme"><span class="track"></span></button>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Identity</h2></div>
        <div class="field">
          <label class="label" for="account-name">Name</label>
          <input id="account-name" v-model="nameInput" type="text" autocomplete="name">
        </div>
        <p><button class="btn" type="button" :disabled="savingName || !nameInput.trim()" @click="saveName">{{ savingName ? 'Saving…' : 'Save name' }}</button></p>
        <p v-if="nameError" class="scope">{{ nameError }}</p>
      </section>
      <!-- Preferred positions: each tap saves the whole set at once (toast),
           and a refused save puts the buttons back as they were. -->
      <section class="sec account-positions">
        <div class="sec-head"><h2>Positions</h2></div>
        <p class="label">Where you like to play</p>
        <div class="pos-picks" role="group" aria-label="Where you like to play">
          <button v-for="position in positionOptions" :key="position" class="pos-pick" :class="{ 'is-on': positions.includes(position) }"
            type="button" :aria-pressed="positions.includes(position).toString()" :disabled="!positionsReady" @click="togglePosition(position)">
            <span v-if="positions.includes(position)" class="tick" aria-hidden="true">✓</span>{{ position }}
          </button>
        </div>
        <p class="scope pos-help">Pick one or more. Only your group admins see them, to balance teams.</p>
        <p v-if="positionsLoadFailed" class="scope">Couldn't load your positions. Reload the page to try again.</p>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Email</h2></div>
        <div class="field account-email">
          <p id="account-email-label" class="label">Email</p>
          <p class="account-email-value" aria-labelledby="account-email-label">{{ email }}</p>
          <p class="note">Changing your email isn't available yet.</p>
        </div>
      </section>
      <section class="sec">
        <div class="sec-head"><h2>Password</h2></div>
        <p class="scope">To choose a new one, we email you a reset link, the same path as forgot password.</p>
        <p v-if="hasPasskeys" class="scope">A password reset also removes your saved passkeys.</p>
        <p><button class="btn btn-ghost" type="button" :disabled="sendingReset || !email" @click="sendReset">{{ sendingReset ? 'Sending…' : 'Email the link' }}</button></p>
        <p v-if="resetNote" class="scope">{{ resetNote }}</p>
      </section>
      <PasskeysV2 v-if="hasPasskeys" @confirming="onChildConfirming" />
      <section class="danger">
        <p class="label">Danger zone</p>
        <h2 class="h2">Delete my account</h2>
        <p>This can't be undone. Your name, email and password are erased, and you're removed from every group. Match history (goals, MOTM titles) stays for the other players, and is no longer tied to an account.</p>
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
      <GroupDetailsForm v-if="isAdmin" :key="groupId" :group="currentGroup" @saved="applyGroupDetails" />
      <section class="sec">
        <div class="sec-head"><h2>Teams</h2></div>
        <p class="scope">Save each team on its own.</p>
        <div v-for="team in teams" :key="team.id" class="team-edit">
          <input v-model="team.name" type="text" :aria-label="`Name of ${team.name}`">
          <button class="btn" type="button" @click="saveTeam(team)">Save</button>
        </div>
        <p v-if="teamNote" class="scope">{{ teamNote }}</p>
      </section>
      <InvitationV2 :group-id="groupId" :code="inviteCode" :share-text="inviteShareText"
        @updated="applyInvitation" @confirming="onChildConfirming" />
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
            <button v-else-if="member.has_account" class="btn btn-ghost" type="button" @click="promote(member)">Make admin</button>
            <button class="btn btn-ghost btn-remove" type="button" :aria-expanded="(removing === member.id).toString()" @click="askRemove(member)">Remove</button>
            <div v-if="removing === member.id" class="confirm-pop" role="dialog">
              <h3>Remove from the group?</h3>
              <p class="ref">{{ displayName(member.name) }} leaves the group. Their match history stays. They need a new invite to come back.</p>
              <div class="confirm-actions actions">
                <button class="btn btn-ghost" type="button" @click="removing = ''">Cancel</button>
                <button class="btn btn-danger" type="button" @click="confirmRemove(member)">Remove</button>
              </div>
            </div>
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
      <div v-if="removing" class="confirm-scrim" @click="removing = ''"></div>
      <div v-if="removing" class="confirm-sheet" role="dialog">
        <h3>Remove from the group?</h3>
        <p class="ref">{{ displayName(removeMemberRow.name) }} leaves the group. Their match history stays. They need a new invite to come back.</p>
        <div class="confirm-actions">
          <button class="btn btn-danger" type="button" @click="confirmRemove(removeMemberRow)">Remove</button>
          <button class="btn btn-ghost" type="button" @click="removing = ''">Cancel</button>
        </div>
      </div>
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
    <AppToast ref="toast" />
  </main>
</template>

<script>
import PasskeysV2 from './PasskeysV2.vue';
import { hasPasskeyAccess } from '@/services/passkeyAccess';
import {
  getPlayerProfile, getMyGroups, getSeasons, setFavoriteGroup, updateMyName,
  deleteMyAccount, clearToken, getToken, getTeamsByGroup, updateTeam, getInviteCode,
  getGroupMembers, updateMemberRole, removeMember, getNotificationPreferences, updateNotificationPreferences,
  forgotPassword, getMyFeatures, getMyPositions, updateMyPositions,
} from '@/services/api';
import { POSITIONS, normalizePositions } from '@/services/v2Positions';
import { actionErrorMessage } from '@/services/v2Match';
import AppToast from '@/components/AppToast.vue';
import { getPushState, enablePush, disablePush, PUSH_STATE, PUSH_FEATURE } from '@/services/pushNotifications';
import { getInstallTracker, INSTALL_STATE, PWA_INSTALL_FEATURE } from '@/services/pwaInstall';
import { buildGroupInviteShareText } from '@/services/shareMessages';
import InvitationV2 from './InvitationV2.vue';
import GroupDetailsForm from './GroupDetailsForm.vue';
import GearLink from './GearLink.vue';

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
  { label: 'Admin', admin: true, items: [
    { key: 'member_joined', label: 'New member joins' },
  ] },
];

function playerIdFromToken() {
  const token = getToken();
  if (!token) return '';
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
    return decoded.player_id || '';
  } catch {
    return '';
  }
}

export default {
  name: 'ProfileV2',
  components: { AppToast, GearLink, GroupDetailsForm, InvitationV2, PasskeysV2 },
  inject: {
    v2Context: { default: () => ({}) },
    openGuide: { default: () => {} },
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
      nameInput: '',
      savingName: false,
      nameError: '',
      positionOptions: POSITIONS,
      positions: [],
      savedPositions: [],
      positionsReady: false,
      positionsLoadFailed: false,
      positionsSeq: 0,
      email: '',
      resetNote: '',
      sendingReset: false,
      deleting: false,
      deletePassword: '',
      deleteError: '',
      teams: [],
      inviteCode: '',
      members: [],
      teamNote: '',
      demoting: '',
      removing: '',
      childConfirming: false,
      roleError: '',
      pushError: '',
      // Hidden features (GET /players/me/features): notifications and the
      // install row only show for players they were enabled for, as on
      // today's profile.
      features: [],
      installState: INSTALL_STATE.UNAVAILABLE,
    };
  },
  computed: {
    // The admin notifications are only listed to someone who is an admin of
    // at least one group.
    preferenceGroups() {
      const admin = this.groups.some((group) => group.role === 'admin');
      return PREFERENCE_GROUPS.filter((group) => admin || !group.admin);
    },
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
    currentGroup() {
      return this.groups.find((group) => group.id === this.groupId);
    },
    inviteShareText() {
      if (!this.inviteCode) return '';
      const inviteUrl = `${window.location.origin}/signup?invite=${this.inviteCode}`;
      return buildGroupInviteShareText({ inviteUrl, groupInviteCode: this.inviteCode });
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
      return this.currentGroup?.role === 'admin';
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
    // Only on or off can be switched: turning on needs the installed app
    // (getPushState says needs-install in a tab), a blocked permission only
    // changes in the settings, and an unsupported browser has nothing to do.
    canTogglePush() {
      return this.pushState === PUSH_STATE.ON || this.pushState === PUSH_STATE.OFF;
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
    removeMemberRow() {
      return this.members.find((member) => member.id === this.removing) || {};
    },
    hasPush() {
      return this.features.includes(PUSH_FEATURE);
    },
    hasPasskeys() {
      return hasPasskeyAccess(this.features);
    },
    hasInstall() {
      return this.features.includes(PWA_INSTALL_FEATURE);
    },
    installOffered() {
      return this.hasInstall && (this.installState === INSTALL_STATE.PROMPT || this.installState === INSTALL_STATE.IOS);
    },
  },
  watch: {
    '$route.fullPath'() {
      this.syncChrome();
      if (this.page === 'settings') this.loadSettings();
      if (this.page === 'notifications') this.loadNotifications();
      if (this.page === 'account') this.loadPositions();
    },
    demoting() {
      this.syncConfirming();
    },
    removing() {
      this.syncConfirming();
    },
    deleting() {
      this.syncConfirming();
    },
  },
  created() {
    this.playerId = playerIdFromToken();
    const tracker = getInstallTracker();
    this.installState = tracker.state();
    this.unsubscribeInstall = tracker.subscribe((state) => {
      this.installState = state;
    });
    this.load();
  },
  beforeUnmount() {
    if (this.unsubscribeInstall) this.unsubscribeInstall();
    if (!this.v2Context) return;
    this.v2Context.hide = false;
    this.v2Context.pageTitle = '';
    this.v2Context.backTo = '';
    this.v2Context.pageAction = null;
    this.v2Context.confirming = false;
  },
  methods: {
    onChildConfirming(open) {
      this.childConfirming = open;
      this.syncConfirming();
    },
    syncConfirming() {
      if (this.v2Context) this.v2Context.confirming = Boolean(this.demoting || this.removing || this.deleting || this.childConfirming);
    },
    install() {
      getInstallTracker().prompt();
    },
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
      // The gear to the group's settings, on the title row; admins only.
      this.v2Context.pageAction = this.page === 'group' && this.isAdmin
        ? { to: `/profile/groups/${this.groupId}/settings`, label: 'Group settings' }
        : null;
    },
    async load() {
      this.loading = true;
      this.failed = false;
      try {
        const [profile, groups, features] = await Promise.all([
          getPlayerProfile(this.season),
          getMyGroups(),
          getMyFeatures().catch(() => []),
        ]);
        this.features = Array.isArray(features) ? features : [];
        this.overall = profile.Overall || {};
        this.perGroup = Array.isArray(profile.PerGroup) ? profile.PerGroup : [];
        this.groups = Array.isArray(groups) ? groups : [];
        this.nameInput = this.overall.Name || '';
        this.email = profile.Email || '';
        if (this.page === 'group') {
          this.seasons = await getSeasons(this.groupId).catch(() => []);
        } else {
          this.seasons = Array.isArray(profile.Seasons) ? profile.Seasons : [];
        }
        this.pushState = await getPushState().catch(() => PUSH_STATE.UNSUPPORTED);
        this.syncChrome();
        if (this.page === 'settings') await this.loadSettings();
        if (this.page === 'notifications') {
          // Reached by URL without the feature: nothing to turn on here.
          if (!this.hasPush) {
            this.$router.replace('/profile');
            return;
          }
          await this.loadNotifications();
        }
        if (this.page === 'account') await this.loadPositions();
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
      if (!this.canTogglePush) return;
      this.pushError = '';
      try {
        this.pushState = this.pushOn ? await disablePush() : await enablePush();
      } catch (error) {
        console.error(error);
        this.pushError = (error.response && error.response.data && error.response.data.error) || error.message || 'Could not change the notifications.';
      }
    },
    async loadPositions() {
      this.positionsLoadFailed = false;
      try {
        const positions = normalizePositions(await getMyPositions());
        this.positions = positions;
        this.savedPositions = positions;
        this.positionsReady = true;
      } catch (error) {
        console.error(error);
        this.positionsReady = false;
        this.positionsLoadFailed = true;
      }
    },
    // Saved at every tap, optimistically: the button changes at once and
    // the whole set is sent. Saves go out one after the other, so the server
    // ends on the latest tap; the server's answer becomes the saved state and
    // a refusal puts back the last saved set with the server's reason. Only
    // the latest tap's answer updates the buttons.
    togglePosition(position) {
      if (!this.positionsReady) return Promise.resolve();
      const picked = new Set(this.positions);
      if (picked.has(position)) picked.delete(position);
      else picked.add(position);
      const next = normalizePositions([...picked]);
      this.positions = next;
      this.positionsSeq += 1;
      const seq = this.positionsSeq;
      const send = async () => {
        try {
          const saved = normalizePositions(await updateMyPositions(next));
          this.savedPositions = saved;
          if (seq !== this.positionsSeq) return;
          this.positions = saved;
          this.$refs.toast?.show('Positions saved.', 'success');
        } catch (error) {
          console.error(error);
          if (seq !== this.positionsSeq) return;
          this.positions = this.savedPositions;
          this.$refs.toast?.show(actionErrorMessage(error, 'Could not save your positions.'), 'error');
        }
      };
      this.positionsQueue = (this.positionsQueue || Promise.resolve()).then(send);
      return this.positionsQueue;
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
    // Same request as the Forgot password page, with the signed-in player's
    // own address; the backend's generic answer is shown as it comes back.
    async sendReset() {
      if (!this.email || this.sendingReset) return;
      this.resetNote = '';
      this.sendingReset = true;
      try {
        const { message } = await forgotPassword(this.email);
        this.resetNote = message;
      } catch (error) {
        this.resetNote = (error.response && error.response.data && error.response.data.error) || 'Could not send the reset link. Please try again.';
      } finally {
        this.sendingReset = false;
      }
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
      const groupId = this.groupId;
      this.applyInvitation(null);
      if (!this.isAdmin) {
        this.$router.replace(`/profile/groups/${groupId}`);
        return;
      }
      const [teams, code, members] = await Promise.all([
        getTeamsByGroup(groupId).catch(() => []),
        getInviteCode(groupId).catch(() => ({ invite_code: '' })),
        getGroupMembers(groupId).catch(() => []),
      ]);
      if (this.page !== 'settings' || this.groupId !== groupId) return;
      this.teams = (Array.isArray(teams) ? teams : []).map((team) => ({
        id: team.id || team.ID,
        name: team.name || team.Name,
        // Team colours aren't shown in v2. The stored value is sent back
        // unchanged on save (the API requires one), for players still on v1.
        colour: team.colour || team.Colour,
      }));
      this.applyInvitation(code);
      this.members = Array.isArray(members) ? members : [];
      this.syncChrome();
    },
    applyInvitation(invitation) {
      this.inviteCode = invitation?.invite_code || invitation?.InviteCode || '';
    },
    applyGroupDetails(updated) {
      this.groups = this.groups.map((group) => group.id === updated.id ? { ...group, ...updated } : group);
    },
    async saveTeam(team) {
      this.teamNote = '';
      try {
        await updateTeam(this.groupId, team.id, team.name.trim(), team.colour);
        this.teamNote = 'Saved.';
      } catch {
        this.teamNote = 'Could not save the team.';
      }
    },
    askDemote(member) {
      this.removing = '';
      this.demoting = member.id;
      this.roleError = '';
    },
    askRemove(member) {
      this.demoting = '';
      this.removing = member.id;
      this.roleError = '';
    },
    // Same as today's roster panel: offered on every row but your own (the
    // backend refuses that one too), behind a confirmation.
    async confirmRemove(member) {
      if (!member || !member.id) return;
      try {
        await removeMember(this.groupId, member.id);
        this.removing = '';
        await this.loadSettings();
      } catch (error) {
        this.removing = '';
        this.roleError = (error.response && error.response.data && error.response.data.error) || 'Could not remove the member.';
      }
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

<style scoped>
.guide-action {
  width: 100%;
  color: inherit;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--v2-rule);
}

.group-description {
  margin-bottom: 8px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.group-website {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 44px;
  max-width: 100%;
  color: inherit;
  font-weight: 600;
  text-decoration: underline;
  overflow-wrap: anywhere;
}

.group-website svg {
  flex: none;
}
</style>
