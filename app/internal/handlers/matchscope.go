package handlers

import (
	"errors"
	"net/http"

	"app/internal/models"
	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The two middlewares below are the third way this codebase resolves the group
// a request acts on, and they exist because neither of the other two fits the
// /matches/:id/... routes:
//
//   - RequireGroupMembership / RequireGroupAdmin resolve the group from a query
//     param, the JSON body, or — failing both — the *caller's own first group*.
//     Used here that would be a genuine vulnerability: the path names a match,
//     so a member of group A could act on group B's match just by supplying
//     their own group id (or none at all).
//   - RequireGroupMembershipByPathParam / RequireGroupAdminByPathParam read the
//     group id straight from the path. These paths carry no group id.
//
// So they derive the group from the match named by the path param, then
// authorize against *that* group — the caller has no say in which group is
// checked.
//
// The middleware stores the authorized group in the Gin context for the handler.

// authorizedGroupIDKey holds the group the caller was authorized against.
const authorizedGroupIDKey = "authorized_group_id"

// RequireGroupMembershipByMatchPathParam authorizes a route whose path param
// paramName holds a *match* id: it loads that match's group and rejects unless
// the authenticated caller is a member of it. On success the group id is put in
// the context for the handler (see authorizedGroupIDFromContext).
//
// A caller who is not a member gets 404 "match not found", not 403. That is the
// convention MatchService.GetMatchDetailsByID and DeleteMatch already
// established — a match id belonging to another group reads as absent rather
// than forbidden — and it matters: a 403 would confirm the match exists, so
// anyone could enumerate other groups' match ids by watching the status code
// flip from 404 to 403.
func RequireGroupMembershipByMatchPathParam(matchService *services.MatchService, membershipService *services.GroupMembershipService, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, groupID, ok := resolveMatchGroupForCaller(c, matchService, paramName)
		if !ok {
			return
		}

		isMember, err := membershipService.IsMember(groupID, playerID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(err))
			return
		}
		if !isMember {
			abortMatchNotFound(c)
			return
		}

		c.Set(authorizedGroupIDKey, groupID)
		c.Next()
	}
}

// RequireGroupAdminByMatchPathParam is the stricter sibling: same match →
// group resolution, but the caller must be an admin of that group.
//
// The two failure modes answer differently, on purpose. A non-member gets the
// same 404 as above (they must not learn the match exists), but a member who
// merely isn't an admin gets 403: they can already read the match through
// GET /matches/:id/details, so there is nothing left to leak and 403 is the
// honest answer — "this exists, you may not do that to it".
func RequireGroupAdminByMatchPathParam(matchService *services.MatchService, membershipService *services.GroupMembershipService, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, groupID, ok := resolveMatchGroupForCaller(c, matchService, paramName)
		if !ok {
			return
		}

		role, err := membershipService.GetRole(groupID, playerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				abortMatchNotFound(c)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(err))
			return
		}
		if role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, messageBody("not an admin of this group"))
			return
		}

		c.Set(authorizedGroupIDKey, groupID)
		c.Next()
	}
}

// resolveMatchGroupForCaller is the shared prologue of the two middlewares
// above: the authenticated player, and the group owning the match named by the
// path param. Like resolveGroupIDForMembership it writes the error response
// itself and returns ok=false.
//
// The match id is parsed before any service call, the same way every handler
// reading an id from the URL does, so a malformed one is a 400 rather than a
// lookup for a nil uuid.
func resolveMatchGroupForCaller(c *gin.Context, matchService *services.MatchService, paramName string) (playerID, groupID uuid.UUID, ok bool) {
	playerID, ok = playerIDFromContext(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, messageBody("missing authentication"))
		return uuid.Nil, uuid.Nil, false
	}

	matchID, err := uuid.Parse(c.Param(paramName))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, messageBody("invalid match id"))
		return uuid.Nil, uuid.Nil, false
	}

	groupID, err = matchService.GetGroupIDByMatchID(matchID)
	if err != nil {
		if errors.Is(err, services.ErrMatchNotFound) {
			abortMatchNotFound(c)
			return uuid.Nil, uuid.Nil, false
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(err))
		return uuid.Nil, uuid.Nil, false
	}

	return playerID, groupID, true
}

// abortMatchNotFound answers with the one message both "no such match" and
// "not your group's match" share. Keeping it in a single place is what keeps
// the two indistinguishable to a caller probing for match ids — two subtly
// different wordings would leak exactly what the shared status code hides.
func abortMatchNotFound(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, errorBody(services.ErrMatchNotFound))
}
