// The API responses the visual tests render. Hand-written rather than captured
// from a running backend, for one reason: `cmd/seed` builds its matches on "the
// last N Sundays" with weighted-random goals, so real seeded data changes every
// day and would make every screenshot a fresh diff. These fixtures are frozen,
// and so is the clock the pages read (see FIXED_NOW).
//
// The shapes are the Go DTOs' JSON, PascalCase included — `MatchWithDetails`,
// `TeamWithPlayers`, `PlayerCustom`, `PointsStandingRow`, `ScorerRow`,
// `MatchRegistrationEntry` in internal/models/customModels.go, and the
// lowercase `GroupWithRole`/`PlayerWithRole`. Getting a key's case wrong here
// would render a blank cell instead of failing, so they are worth checking
// against the structs rather than against memory.

// Friday 4 September 2026, 10:00 Paris — two days before the scheduled match
// below kicks off, and inside the 2026-2027 season. Every page under test reads
// `Date.now()` (the sign-up panel derives its whole state from it), so the tests
// pin it rather than letting the calendar decide what the screenshots contain.
const FIXED_NOW = new Date('2026-09-04T10:00:00+02:00');

const GROUP_ID = '11111111-1111-4111-8111-111111111111';
const OTHER_GROUP_ID = '22222222-2222-4222-8222-222222222222';

// The player the fake JWT belongs to. Present in the sign-up list on purpose:
// it is what renders the "you" badge, so the tests cover that branch.
const CURRENT_PLAYER_ID = '33333333-3333-4333-8333-333333333333';

const TEAM_RED = {
  ID: 'aaaaaaaa-0000-4000-8000-000000000001',
  Name: 'Les Rouges',
  Colour: 'red',
};
const TEAM_BLUE = {
  ID: 'aaaaaaaa-0000-4000-8000-000000000002',
  Name: 'Les Bleus',
  Colour: 'blue',
};

const player = (n, name, goals) => ({
  ID: `bbbbbbbb-0000-4000-8000-0000000000${String(n).padStart(2, '0')}`,
  Name: name,
  GoalNumber: goals,
});

const groups = [
  { id: GROUP_ID, name: 'Calciotto Milano', role: 'admin', is_favorite: true },
  { id: OTHER_GROUP_ID, name: 'Sunday League', role: 'member', is_favorite: false },
];

// Ascending, as the backend returns them — MatchesAndStandings preselects the
// last one, so "2026-2027" is what every screenshot is scoped to.
const seasons = ['2025-2026', '2026-2027'];

// A played match, both teams filled: the ordinary case, and the only one the
// standings count.
const playedMatch = {
  ID: 'cccccccc-0000-4000-8000-000000000001',
  GroupID: GROUP_ID,
  Date: '2026-08-30',
  Teams: [
    {
      ...TEAM_BLUE,
      Score: 4,
      Players: [
        player(1, 'Marco Rossi', 2),
        player(2, 'Luca Bianchi', 1),
        player(3, 'Thibaut Fabre', 1),
        player(4, 'Giovanni Esposito', 0),
      ],
    },
    {
      ...TEAM_RED,
      Score: 6,
      Players: [
        player(5, 'Andrea Conti', 3),
        player(6, 'Matteo Ricci', 2),
        player(7, 'Federico Greco', 1),
        player(8, 'Alessandro Moretti', 0),
      ],
    },
  ],
};

const olderPlayedMatch = {
  ID: 'cccccccc-0000-4000-8000-000000000002',
  GroupID: GROUP_ID,
  Date: '2026-09-01',
  Teams: [
    {
      ...TEAM_BLUE,
      Score: 3,
      Players: [player(1, 'Marco Rossi', 1), player(6, 'Matteo Ricci', 2)],
    },
    {
      ...TEAM_RED,
      Score: 3,
      Players: [player(5, 'Andrea Conti', 2), player(3, 'Thibaut Fabre', 1)],
    },
  ],
};

// A scheduled match with sign-ups open: no roster yet, which is its normal
// state — the teams are composed only once the list is closed.
const scheduledOpenMatch = {
  ID: 'cccccccc-0000-4000-8000-000000000003',
  GroupID: GROUP_ID,
  // Derived by the backend from ScheduledAt's calendar day, and equal to it
  // here for the same reason: they can never disagree.
  Date: '2026-09-06',
  ScheduledAt: '2026-09-06T20:30:00+02:00',
  RegistrationOpensAt: '2026-09-01T18:00:00+02:00',
  MaxPlayers: 16,
  RegistrationCount: 18,
  Teams: [
    { ...TEAM_BLUE, Score: 0, Players: [] },
    { ...TEAM_RED, Score: 0, Players: [] },
  ],
};

// The same match after the admin closed the list — the state in which
// "Fill teams from sign-ups" is offered.
const scheduledClosedMatch = {
  ...scheduledOpenMatch,
  ID: 'cccccccc-0000-4000-8000-000000000004',
  RegistrationsClosedAt: '2026-09-04T09:00:00+02:00',
};

// Newest last, as GetMatchesDetails returns them (ordered by date).
const matches = [olderPlayedMatch, playedMatch, scheduledOpenMatch];

// 18 sign-ups against MaxPlayers: 16 → the last two are the waiting list, which
// is derived from this order rather than stored. The current player sits at #14
// so the "you" badge lands in the confirmed list.
const registrationNames = [
  'Andrea Conti', 'Marco Rossi', 'Luca Bianchi', 'Matteo Ricci',
  'Federico Greco', 'Alessandro Moretti', 'Giovanni Esposito', 'Davide Fontana',
  'Simone Barbieri', 'Riccardo Gallo', 'Stefano Marino', 'Antonio Villa',
  'Paolo Ferrari', 'Thibaut Fabre', 'Nicola Sanna', 'Emanuele Costa',
  'Gabriele Longo', 'Vincenzo Pace',
];

const registrations = registrationNames.map((name, index) => ({
  PlayerID: name === 'Thibaut Fabre'
    ? CURRENT_PLAYER_ID
    : `dddddddd-0000-4000-8000-0000000000${String(index + 1).padStart(2, '0')}`,
  Name: name,
  Position: index + 1,
  IsWaiting: index + 1 > scheduledOpenMatch.MaxPlayers,
  RegisteredAt: new Date(Date.UTC(2026, 8, 1, 16, 5 * index)).toISOString(),
}));

// One row with IsMember false on purpose: it is what renders the
// "(left the group)" tag, a branch no unit test can show visually.
const pointsStandings = [
  { PlayerID: 'e1', Name: 'Andrea Conti', Played: 9, Won: 6, Drawn: 2, Lost: 1, GoalsFor: 14, Points: 20, IsMember: true },
  { PlayerID: 'e2', Name: 'Marco Rossi', Played: 9, Won: 5, Drawn: 2, Lost: 2, GoalsFor: 11, Points: 17, IsMember: true },
  { PlayerID: 'e3', Name: 'Matteo Ricci', Played: 8, Won: 4, Drawn: 3, Lost: 1, GoalsFor: 9, Points: 15, IsMember: true },
  { PlayerID: CURRENT_PLAYER_ID, Name: 'Thibaut Fabre', Played: 9, Won: 4, Drawn: 1, Lost: 4, GoalsFor: 7, Points: 13, IsMember: true },
  { PlayerID: 'e5', Name: 'Luca Bianchi', Played: 7, Won: 3, Drawn: 1, Lost: 3, GoalsFor: 5, Points: 10, IsMember: true },
  { PlayerID: 'e6', Name: 'Davide Fontana', Played: 6, Won: 2, Drawn: 1, Lost: 3, GoalsFor: 4, Points: 7, IsMember: false },
];

const topScorers = [
  { PlayerID: 'e1', Name: 'Andrea Conti', Played: 9, Goals: 14, IsMember: true },
  { PlayerID: 'e2', Name: 'Marco Rossi', Played: 9, Goals: 11, IsMember: true },
  { PlayerID: 'e3', Name: 'Matteo Ricci', Played: 8, Goals: 9, IsMember: true },
  { PlayerID: CURRENT_PLAYER_ID, Name: 'Thibaut Fabre', Played: 9, Goals: 7, IsMember: true },
  { PlayerID: 'e6', Name: 'Davide Fontana', Played: 6, Goals: 4, IsMember: false },
];

// GET /groups/:id/players — PlayerWithRole, so lowercase keys with `role`
// alongside. The credential-less entry is the admin-created "ghost player" the
// roster shows an "Invite" action for.
const groupMembers = [
  { id: CURRENT_PLAYER_ID, name: 'Thibaut Fabre', email: 'thibaut@example.com', role: 'admin' },
  { id: 'e1', name: 'Andrea Conti', email: 'andrea@example.com', role: 'member' },
  { id: 'e2', name: 'Marco Rossi', role: 'member' },
];

module.exports = {
  FIXED_NOW,
  GROUP_ID,
  CURRENT_PLAYER_ID,
  groups,
  seasons,
  matches,
  playedMatch,
  scheduledOpenMatch,
  scheduledClosedMatch,
  registrations,
  pointsStandings,
  topScorers,
  groupMembers,
};
