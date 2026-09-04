package handlers

import (
	"errors"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MatchVoteHandler exposes a match's Man of the Match vote/tally over HTTP.
//
// Like MatchRegistrationHandler, every route is authorized by
// RequireGroupMembershipByMatchPathParam: the group is derived from the match
// named in the path, never supplied by the caller (see matchscope.go). Unlike
// the sign-up feature, there is no admin-only route here at all — voter
// eligibility is deliberately broader than "played in the match" (any group
// member can judge), and there is no close/reopen concept for voting, so
// nothing in this feature needs RequireGroupAdminByMatchPathParam.
type MatchVoteHandler struct {
	Service *services.MatchVoteService
}

func NewMatchVoteHandler(service *services.MatchVoteService) *MatchVoteHandler {
	return &MatchVoteHandler{Service: service}
}

// voteRequest binds POST /matches/:id/votes' body. Snake_case, matching this
// codebase's other dedicated request structs (e.g. createMatchRequest) rather
// than models.MatchVote's own json tags.
type voteRequest struct {
	VotedForID uuid.UUID `json:"voted_for_id" binding:"required"`
}

// Vote casts or replaces the *caller's* vote — the voter always comes from
// the JWT, never the body, the same "no acting on someone else's behalf"
// rule as MatchRegistrationHandler.Register. Unlike Register, a repeated call
// is not a conflict: MatchVoteService.Vote is an upsert, so calling this
// again after already voting simply changes the vote. The response is the
// resulting tally (the same shape ListVotes returns) so the client can
// render the updated counts without a second request.
func (h *MatchVoteHandler) Vote(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	voterID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var req voteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voted_for_id is required"})
		return
	}

	if err := h.Service.Vote(matchID, voterID, req.VotedForID); err != nil {
		respondMatchVoteError(c, err)
		return
	}

	summary, err := h.Service.ListVotes(matchID, voterID)
	if err != nil {
		respondMatchVoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// Unvote removes the caller's own vote for the match, if any. A no-op
// success when they had not voted — the same "a retried request must not
// fail for nothing" reasoning as ReopenRegistrations — so this never answers
// 404 for that reason alone.
func (h *MatchVoteHandler) Unvote(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	voterID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	if err := h.Service.Unvote(matchID, voterID); err != nil {
		respondMatchVoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unvoted": true})
}

// ListVotes returns the tally plus which player, if any, the caller has voted
// for. Open to any member of the match's group — seeing the tally isn't
// privileged, the same reasoning as MatchRegistrationHandler.ListRegistrations.
func (h *MatchVoteHandler) ListVotes(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	callerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	summary, err := h.Service.ListVotes(matchID, callerID)
	if err != nil {
		respondMatchVoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// respondMatchVoteError maps MatchVoteService's sentinels to status codes.
// Self-vote and not-on-roster are both malformed requests (400) rather than
// state conflicts (409): unlike a sign-up window opening or closing, whether
// a candidate is on the roster and whether the voter is themselves are true
// or false regardless of timing, so there is nothing to retry into validity.
func respondMatchVoteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrMatchNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrCannotVoteForSelf),
		errors.Is(err, services.ErrVotedForPlayerNotOnRoster):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
