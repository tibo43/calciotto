package handlers

import (
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// groupIDFailure is the one way group resolution can fail, expressed as data
// rather than as an already-written response: resolveScopedGroupID is shared
// between a *handler* (which answers with c.JSON and then returns) and a
// *middleware* (which has to abort the chain), so it cannot pick the writing
// style itself. Returning the status and message instead keeps both spellings
// of the same answer derived from one place.
type groupIDFailure struct {
	status  int
	message string
}

// resolveScopedGroupID is the single implementation of "which group does this
// request act on", shared by resolveGroupID (handlers) and
// resolveGroupIDForMembership (middlewares).
//
// Sharing it is a correctness property, not tidiness. DELETE /matches/:id
// resolves its group twice — once in requireGroupAdmin, to decide whether the
// caller may act, and once in MatchHandler.DeleteMatch, to scope the delete —
// and those two answers have to be the same group or an admin of group A could
// have their delete applied to group B. They used to be two independent copies
// of this logic that merely happened to agree for every request; now they
// agree by construction, and a future change to the resolution rules can only
// be made in one place.
//
// The order is: group_id from the query string, then — only when fromBody is
// supplied — the same field out of the JSON body, then a group the
// authenticated caller (AuthMiddleware sets "player_id" in the context)
// actually belongs to, so existing callers like the current frontend keep
// working without passing a group_id explicitly.
//
// The player is looked up lazily, at that last step, and not before: a request
// that names its group in the query string is answered without needing one at
// all, which is the behaviour both entry points already had.
//
// It deliberately does NOT fall back to the old sort-by-random-UUID default
// (GroupService.GetDefaultGroup, since removed), which had no relation to who
// the caller is — that mismatch let anyone flip every existing user's default
// group (and thus their access) just by creating an unrelated second group.
func resolveScopedGroupID(c *gin.Context, membershipService *services.GroupMembershipService, fromBody func(*gin.Context) (uuid.UUID, bool)) (uuid.UUID, *groupIDFailure) {
	if raw := c.Query("group_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, &groupIDFailure{status: http.StatusBadRequest, message: "invalid group_id"}
		}
		return parsed, nil
	}

	if fromBody != nil {
		if id, ok := fromBody(c); ok {
			return id, nil
		}
	}

	playerID, ok := playerIDFromContext(c)
	if !ok {
		return uuid.Nil, &groupIDFailure{status: http.StatusUnauthorized, message: "missing authentication"}
	}

	group, err := membershipService.GetFirstGroupForPlayer(playerID)
	if err != nil {
		return uuid.Nil, &groupIDFailure{status: http.StatusNotFound, message: "authenticated player does not belong to any group"}
	}
	return group.ID, nil
}

// resolveGroupID is the handler-side entry point: the shared resolution above
// with no body step (a read handler's group travels in the query string), and
// the failure written with c.JSON since the handler is the end of the chain
// and simply returns afterwards. This must only be used behind AuthMiddleware.
// On failure it writes the error response itself and returns ok=false.
func resolveGroupID(c *gin.Context, membershipService *services.GroupMembershipService) (id uuid.UUID, ok bool) {
	groupID, failure := resolveScopedGroupID(c, membershipService, nil)
	if failure != nil {
		c.JSON(failure.status, gin.H{"error": failure.message})
		return uuid.Nil, false
	}
	return groupID, true
}
