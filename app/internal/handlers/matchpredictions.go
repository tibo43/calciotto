package handlers

import (
	"errors"
	"net/http"

	"app/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MatchPredictionHandler exposes a match's win/draw/loss predictions and
// Elo-derived odds over HTTP.
//
// Like MatchVoteHandler, every route is authorized by
// RequireGroupMembershipByMatchPathParam: the group is derived from the
// match named in the path, never supplied by the caller (see matchscope.go).
// There is no admin-only route at all, and — unlike Man of the Match voting
// — no roster-eligibility check either: any group member may predict a
// match's result, whether or not they are themselves playing in it.
type MatchPredictionHandler struct {
	Service *services.MatchPredictionService
}

func NewMatchPredictionHandler(service *services.MatchPredictionService) *MatchPredictionHandler {
	return &MatchPredictionHandler{Service: service}
}

// predictRequest binds POST /matches/:id/predictions' body. PredictedTeamID
// is a *uuid.UUID with no binding:"required": an explicit JSON null is how a
// draw is predicted, so "field omitted" has to stay distinguishable from "a
// deliberate draw pick" — binding:"required" would reject the very value a
// draw prediction needs to send.
type predictRequest struct {
	PredictedTeamID *uuid.UUID `json:"predicted_team_id"`
}

// Predict casts or replaces the caller's own prediction — the player always
// comes from the JWT, never the body, the same "no acting on someone else's
// behalf" rule as MatchVoteHandler.Vote. Like it, this is an upsert: a
// repeat call simply changes the prediction rather than conflicting.
// Responds with the resulting summary (odds included) so the client can
// render the update without a second request.
func (h *MatchPredictionHandler) Predict(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	playerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	var req predictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.Service.Predict(matchID, playerID, req.PredictedTeamID); err != nil {
		respondMatchPredictionError(c, err)
		return
	}

	summary, err := h.Service.ListPredictions(matchID, playerID)
	if err != nil {
		respondMatchPredictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// ListPredictions returns every member's prediction for the match, the odds
// they were/are made against, and the caller's own prediction. Open to any
// member — seeing predictions isn't privileged, the same reasoning as
// MatchVoteHandler.ListVotes.
func (h *MatchPredictionHandler) ListPredictions(c *gin.Context) {
	matchID, ok := matchIDFromPath(c)
	if !ok {
		return
	}
	callerID, ok := playerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication"})
		return
	}

	summary, err := h.Service.ListPredictions(matchID, callerID)
	if err != nil {
		respondMatchPredictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

// respondMatchPredictionError maps MatchPredictionService's sentinels to
// status codes. ErrPredictionsNotAvailable and ErrPredictedTeamNotInMatch
// are both malformed-request 400s rather than state conflicts: whether a
// match is scheduled/composed, and whether a team id belongs to it, are true
// or false regardless of timing, so there is nothing to retry into validity.
// ErrPredictionsClosed, on the other hand, IS a timing-dependent state
// conflict — the same 409 convention as ErrRegistrationsClosed/
// ErrVotingClosed — since the identical request would have succeeded a
// moment earlier.
func respondMatchPredictionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrMatchNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPredictionsNotAvailable),
		errors.Is(err, services.ErrPredictedTeamNotInMatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrPredictionsClosed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
