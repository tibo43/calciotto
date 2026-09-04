package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Modèle de base pour les champs communs
type BaseModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
}

// BeforeCreate est un hook GORM qui s'exécute avant la création d'un enregistrement
func (bm *BaseModel) BeforeCreate(tx *gorm.DB) error {
	bm.ID = uuid.New() // Génération d'un nouvel UUID
	return nil
}

// Player représente un joueur. Email/PasswordHash restent nuls tant que le
// joueur n'a pas "réclamé" son compte via AuthService.Signup — un Player
// peut donc exister sans jamais être associé à des identifiants de connexion.
type Player struct {
	BaseModel
	Name             string        `gorm:"type:string" json:"name"`
	Email            *string       `gorm:"type:string;uniqueIndex" json:"email,omitempty"`
	PasswordHash     string        `gorm:"type:string" json:"-"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:PlayerID"`
}

// PasswordResetToken matérialise un lien "mot de passe oublié" à usage unique,
// émis par AuthService.ForgotPassword et consommé par ResetPassword.
//
// TokenHash contient le SHA-256 (hex) du token brut, jamais le token lui-même :
// une fuite de cette table ne permet donc pas de forger un lien valide. Le hash
// est volontairement rapide, contrairement à Player.PasswordHash qui utilise
// bcrypt — un token de reset est généré aléatoirement (32 octets d'entropie),
// donc hors de portée d'une attaque par dictionnaire ou par force brute, là où
// un mot de passe choisi par un humain a besoin du coût délibéré de bcrypt.
// bcrypt n'apporterait rien ici et rendrait chaque vérification inutilement
// lente.
//
// UsedAt reste nil tant que le lien n'a pas servi : un token est valide s'il
// existe, qu'il n'est pas expiré (ExpiresAt) et qu'il n'a pas déjà été
// consommé.
type PasswordResetToken struct {
	BaseModel
	PlayerID  uuid.UUID  `gorm:"type:uuid;index" json:"player_id"`
	TokenHash string     `gorm:"type:string;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// Group représente un groupe de joueurs (ex: un club, une salle). Team et
// Match sont scopés par groupe ; un Player peut appartenir à plusieurs
// groupes via GroupMembership.
//
// InviteCode est le secret partagé qui permet à un joueur de rejoindre le
// groupe (POST /groups/join) : il porte donc `json:"-"` pour ne jamais
// s'échapper via les routes publiques GET /groups et GET /groups/:id — seul
// GET /groups/:id/invite-code, réservé aux membres, le renvoie explicitement.
type Group struct {
	BaseModel
	Name       string `gorm:"type:string" json:"name"`
	InviteCode string `gorm:"type:string;uniqueIndex" json:"-"`
}

// Team représente une équipe. Chaque groupe a exactement deux équipes,
// créées à la création du groupe à partir des deux noms/couleurs fournis par
// l'admin qui le crée — voir GroupService.CreateGroup. Name est le nom
// affiché (ex: "Les Rouges"), distinct de Colour qui reste la couleur du
// maillot/swatch utilisée par le frontend.
type Team struct {
	BaseModel
	GroupID          uuid.UUID     `gorm:"type:uuid;index" json:"group_id"`
	Name             string        `gorm:"type:string" json:"name"`
	Colour           string        `gorm:"type:string" json:"colour"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:TeamID"`
}

// Match représente un match, scopé à un groupe.
//
// Scheduling is optional and purely additive: a match created without any of
// the four fields below behaves exactly as it always did (an admin records a
// match that has already been played). That is why all four are nullable —
// no data migration is needed for matches created before the feature
// existed. A match is scheduled iff ScheduledAt is set; use IsScheduled
// rather than scattering nil checks through the callers.
//
// Date stays load-bearing and is deliberately *not* replaced by ScheduledAt:
// SeasonOf derives the season from it, the matches list is ordered by it, and
// its "YYYY-MM-DD" JSON shape is the existing API contract. The two can never
// disagree because MatchService.CreateMatch derives Date from ScheduledAt's
// calendar day instead of letting a caller supply both — a single write path.
//
// RegistrationsClosedAt is a nullable timestamp rather than a bool because an
// admin closes sign-ups manually (in order to then compose the teams) and can
// re-open them to recover from a mis-click: NULL means open, a value records
// when they were closed. It is not the only thing that closes sign-ups,
// though — ScheduledAt is a hard backstop, see RegistrationWindowError.
type Match struct {
	BaseModel
	GroupID uuid.UUID `gorm:"type:uuid;index" json:"group_id"`
	Date    Date      `gorm:"type:date" json:"date"`
	// ScheduledAt is the kick-off date *and time* (unlike Date, which is a
	// calendar day only), hence timestamptz: a 21:00 Paris kick-off has to
	// survive a round-trip through a server running in another zone.
	ScheduledAt           *time.Time    `gorm:"type:timestamptz" json:"scheduled_at,omitempty"`
	RegistrationOpensAt   *time.Time    `gorm:"type:timestamptz" json:"registration_opens_at,omitempty"`
	RegistrationsClosedAt *time.Time    `gorm:"type:timestamptz" json:"registrations_closed_at,omitempty"`
	MaxPlayers            *int          `json:"max_players,omitempty"`
	// CreatedAt is the moment this row was logged — GORM's usual
	// auto-populate-on-create convention, like MatchRegistration.CreatedAt and
	// MatchVote.CreatedAt. It exists on this model specifically as the Man of
	// the Match voting window's anchor for a match with no kick-off at all: an
	// unscheduled match has no better proxy for "when it was played" than the
	// moment an admin recorded it, which in practice is normally right after
	// the game (see MatchVoteService.VotingWindowError). Added after `matches`
	// already had rows, unlike the two structs above — see connect.go's
	// backfill for why that matters here and didn't for RegistrationOpensAt.
	CreatedAt        time.Time     `json:"created_at"`
	TeamCompositions []MatchPlayer `gorm:"foreignKey:MatchID"`
}

// IsScheduled reports whether this is a scheduled match — one with a kick-off
// time players can sign up for — as opposed to a plain match recorded after
// the fact. MatchService.CreateMatch validates scheduling as all-or-nothing,
// so ScheduledAt being set implies RegistrationOpensAt and MaxPlayers are set
// too.
func (m Match) IsScheduled() bool {
	return m.ScheduledAt != nil
}

// MatchRegistration is one player's sign-up for a scheduled match. It is
// deliberately *not* a MatchPlayer: a MatchPlayer carries a TeamID and a goal
// count, neither of which exists yet when a player merely says they will come,
// and — more importantly — ComputePointsStandings treats any match whose two
// teams both have players as played, so recording sign-ups as MatchPlayer rows
// would make next Sunday's match appear in the standings as a 0-0 draw.
//
// The waiting list is derived, never stored: no status column, no promotion
// job. Sign-ups are ordered by CreatedAt (auto-filled by GORM, and therefore
// the key that *defines* the waiting list), the first Match.MaxPlayers of them
// are confirmed and the rest are waiting — see ComputeRegistrationPositions.
// A withdrawal is then a plain row delete which mechanically promotes the next
// player, with no race to lose, and lowering MaxPlayers rolls the tail of the
// list into the waiting list for free.
//
// The composite unique index is what makes a double-clicked "Participate"
// harmless: a player holds at most one sign-up per match.
type MatchRegistration struct {
	BaseModel
	MatchID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_match_registration_match_player" json:"match_id"`
	PlayerID  uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_match_registration_match_player" json:"player_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MatchVote is one group member's vote for who they think deserves the Man
// of the Match award for a match with a composed roster (either an ordinary
// already-recorded match, or a scheduled one once teams have been filled in).
//
// Unlike MatchRegistration.Register (a one-shot action that rejects a
// duplicate with ErrAlreadyRegistered), casting a vote is deliberately an
// upsert: a player changing their mind about who deserves MOTM is the normal
// case, not a mistake to refuse — see MatchVoteService.Vote. The composite
// unique index on (match_id, voter_id) is therefore not "at most one sign-up"
// the way MatchRegistration's is, but "exactly one *current* vote per voter
// per match", enforced the same way.
//
// There is no award or "winner" column anywhere: the MOTM award for a match
// is derived, never stored, by tallying VotedForID per match and taking
// whoever has the most votes — ties included, with no arbitrary tie-break
// (see MatchVoteService.ComputeMotmWinners). That mirrors the waiting list's
// own "derive it, don't store it" choice on MatchRegistration.
type MatchVote struct {
	BaseModel
	MatchID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_match_vote_match_voter" json:"match_id"`
	VoterID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_match_vote_match_voter" json:"voter_id"`
	VotedForID uuid.UUID `gorm:"type:uuid" json:"voted_for_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// MatchPlayer représente la composition d'une équipe pour un match.
type MatchPlayer struct {
	BaseModel
	MatchID     uuid.UUID `gorm:"type:uuid" json:"match_id"`
	TeamID      uuid.UUID `gorm:"type:uuid" json:"team_id"`
	PlayerID    uuid.UUID `gorm:"type:uuid" json:"player_id"`
	GoalsScored int       `gorm:"type:int" json:"goals_scored"`
}

// GroupMembership est la table de jointure many-to-many entre Player et
// Group : un joueur peut appartenir à plusieurs groupes. CreatedAt (auto-rempli
// par GORM à la création) permet de déterminer quel groupe un joueur a
// rejoint en premier — l'ID d'un groupe est un UUID aléatoire (v4), donc trié
// par ID ne dit rien sur l'ordre d'appartenance ou de création.
//
// Role distingue les administrateurs du groupe (RoleAdmin) des simples
// membres (RoleMember) — c'est une simple string plutôt qu'un type enum Go
// dédié, comme Team.Colour ci-dessus : le repo n'a pas de convention pour les
// enums et deux valeurs constantes suffisent ici.
//
// Un groupe peut avoir *plusieurs* administrateurs : le créateur en est le
// premier (POST /groups), et tout admin peut en promouvoir d'autres via
// PATCH /groups/:id/members/:playerId/role. Le seul invariant maintenu est
// qu'un groupe qui a au moins un membre a toujours au moins un admin — il est
// défendu des deux côtés : GroupMembershipService.LeaveGroup promeut le plus
// ancien membre restant si le dernier admin s'en va, et UpdateMemberRole
// refuse de rétrograder le dernier admin (ErrLastAdmin).
//
// Le rôle gouverne aujourd'hui : retirer un membre (DELETE
// /groups/:id/members/:playerId), changer le rôle d'un membre, créer un match
// (POST /matches) et modifier ses scores (PUT /matches/:id) — voir
// RequireGroupAdmin / RequireGroupAdminByPathParam dans
// internal/handlers/groupadmin.go. Lire les matchs et les classements reste
// ouvert à tout membre.
// IsFavorite marks which one of a player's groups resolveActiveGroup()
// (frontend) lands on by default — on a fresh device, after logging in, or
// whenever the locally-stored active group doesn't match anything (the
// player left it, or this browser belongs to a different account). Exactly
// one of a player's memberships carries it at any time, as long as they
// belong to at least one group at all: AddPlayerToGroupWithRole grants it to
// a player's very first membership automatically, GroupMembershipService.
// SetFavoriteGroup is the only way to move it elsewhere, and LeaveGroup/
// RemoveMember reassign it to the oldest remaining membership if the one
// being deleted held it — the invariant is enforced service-side, not by a
// DB constraint, the same way the "a group always has an admin" invariant is.
type GroupMembership struct {
	BaseModel
	GroupID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"group_id"`
	PlayerID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_group_membership_group_player" json:"player_id"`
	Role       string    `gorm:"type:string;not null;default:member" json:"role"`
	IsFavorite bool      `gorm:"not null;default:false" json:"-"`
	CreatedAt  time.Time `json:"-"`
}

// RoleAdmin et RoleMember sont les deux seules valeurs valides de
// GroupMembership.Role. Le créateur d'un groupe (POST /groups) devient
// RoleAdmin ; tout autre joueur ajouté (POST /groups/join, POST
// /groups/:id/players) devient RoleMember, et peut ensuite être promu par un
// admin existant (PATCH /groups/:id/members/:playerId/role).
//
// RoleAdmin valait "owner" avant le passage au modèle multi-admin : les
// lignes déjà en base sont réécrites par la migration ponctuelle de
// pkg/database.InitDB, AutoMigrate ne touchant jamais aux données.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)
