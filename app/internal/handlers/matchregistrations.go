package handlers

import (
	"errors"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MatchRegistrationHandler exposes a scheduled match's sign-up list over HTTP.
//
// It takes no GroupMembershipService, unlike MatchHandler: every one of its
// routes is authorized by RequireGroupMembershipByMatchPathParam or
// RequireGroupAdminByMatchPathParam, which resolve the group from the match in
// the path and publish it in the context, so there is nothing left for the
// handler to resolve or check.
type MatchRegistrationHandler struct {
	Service *services.MatchRegistrationService
}

func NewMatchRegistrationHandler(service *services.MatchRegistrationService) *MatchRegistrationHandler {
	return &MatchRegistrationHandler{Service: service}
}

// Register signs the *caller* up for the match. The player always comes from
// the JWT (playerIDFromContext), never from the body or the path: there is no
// "register someone else" capability in this feature, and taking a player id
// from the request would silently create one.
//
// It answers with the caller's resulting entry rather than a bare
// {"registered": true}, because reaching Match.MaxPlayers is not an error here
// — the surplus sign-up succeeds and lands on the waiting list — so this
// response is the only place the client can learn "you are #17, you are on the
// bench". Anything less would force a second request just to render the button
// that was clicked.
func (h *MatchRegistrationHandler) Register(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.Service.Register(matchID, playerID); err != nil {
		respondMatchRegistrationError(c, err)
		return
	}

	// The position is read back from the authoritative ordered list rather than
	// guessed from a count: position and waiting status are derived from the
	// whole list (see ComputeRegistrationPositions), and the list is the one
	// thing that knows the order the database actually stored.
	entries, err := h.Service.ListRegistrations(matchID)
	if err != nil {
		respondMatchRegistrationError(c, err)
		return
	}
	for _, entry := range entries {
		if entry.PlayerID == playerID {
			c.JSON(http.StatusOK, entry)
			return
		}
	}

	// Unreachable unless the sign-up vanished between the two calls, which
	// would mean a concurrent withdrawal by the same player — reported rather
	// than papered over with a fabricated entry.
	c.JSON(http.StatusInternalServerError, gin.H{"error": "registration was created but could not be read back"})
}

// Unregister withdraws the caller's own sign-up — the only sign-up they can
// touch, which is why the route has no /me suffix: a suffix would imply an
// alternative target that does not exist.
func (h *MatchRegistrationHandler) Unregister(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.Service.Unregister(matchID, playerID); err != nil {
		respondMatchRegistrationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"unregistered": true})
}

// ListRegistrations returns the ordered sign-up list, each entry tagged with
// its position and whether it is on the waiting list. Open to any member of the
// match's group: seeing who is coming isn't privileged, only closing the list
// is.
func (h *MatchRegistrationHandler) ListRegistrations(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}

	entries, err := h.Service.ListRegistrations(matchID)
	if err != nil {
		respondMatchRegistrationError(c, err)
		return
	}
	// The service already normalizes an empty list away from nil, but the
	// guard is kept for the same reason GetMatchesDetails keeps its own: a
	// `null` here would force every client to special-case it.
	if entries == nil {
		entries = []models.MatchRegistrationEntry{}
	}

	c.JSON(http.StatusOK, entries)
}

// CloseRegistrations freezes the roster so an admin can compose the teams.
func (h *MatchRegistrationHandler) CloseRegistrations(c *gin.Context) {
	matchID, groupID, ok := matchAndAuthorizedGroup(c)
	if !ok {
		return
	}

	if err := h.Service.CloseRegistrations(matchID, groupID); err != nil {
		respondMatchRegistrationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"closed": true})
}

// ReopenRegistrations undoes a mis-clicked close.
func (h *MatchRegistrationHandler) ReopenRegistrations(c *gin.Context) {
	matchID, groupID, ok := matchAndAuthorizedGroup(c)
	if !ok {
		return
	}

	if err := h.Service.ReopenRegistrations(matchID, groupID); err != nil {
		respondMatchRegistrationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"reopened": true})
}

// matchIDFromPath parses the :id path param, answering 400 before any service
// call — the same discipline every handler in this package applies to an id
// read from the URL.
func matchIDFromPath(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return uuid.Nil, false
	}
	return id, true
}

// matchAndAuthorizedGroup serves the two admin actions, which are scoped to a
// group as well as a match. The group is taken from the context, where
// RequireGroupAdminByMatchPathParam put the one it authorized — deliberately
// not re-resolved here, so the handler cannot possibly act on a different
// group than the one the caller was checked against.
func matchAndAuthorizedGroup(c *gin.Context) (matchID, groupID uuid.UUID, ok bool) {
	matchID, ok = matchIDFromPath(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	groupID, ok = authorizedGroupIDFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "route is missing its group authorization middleware"})
		return uuid.Nil, uuid.Nil, false
	}

	return matchID, groupID, true
}

// respondMatchRegistrationError maps MatchRegistrationService's sentinels to
// status codes, in one place because all five handlers share the same
// vocabulary.
//
// The 409s are the state conflicts: the request is well-formed and the caller
// is entitled to make it, but the match (or the caller's own sign-up) is not in
// the state it needs. ErrMatchNotScheduled is a 400 instead, because a match
// with no schedule has no sign-up list at all — asking it to accept one is a
// malformed request, not a moment to retry.
func respondMatchRegistrationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrMatchNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrMatchNotScheduled):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrRegistrationsNotOpenYet),
		errors.Is(err, services.ErrRegistrationsClosed),
		errors.Is(err, services.ErrAlreadyRegistered),
		errors.Is(err, services.ErrNotRegistered):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
