package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/domain"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/realtime"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/repository"
	"github.com/Bodhisattwa-Ghosh/HCL-Project/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const userContextKey = "authenticatedUser"

type Handler struct {
	auth           *service.AuthService
	polls          *service.PollService
	broker         *realtime.Broker
	cookieSecure   bool
	cookieSameSite http.SameSite
}

func New(auth *service.AuthService, polls *service.PollService, broker *realtime.Broker, cookieSecure bool, cookieSameSite http.SameSite) *Handler {
	return &Handler{auth: auth, polls: polls, broker: broker, cookieSecure: cookieSecure, cookieSameSite: cookieSameSite}
}

func (h *Handler) Register(router *gin.Engine) {
	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	api.POST("/auth/signup", h.signUp)
	api.POST("/auth/login", h.login)
	api.GET("/auth/me", h.requireUser(), h.me)

	api.GET("/polls", h.requireUser(), h.listPolls)
	api.POST("/polls", h.requireUser(), h.createPoll)
	api.GET("/polls/:slug", h.getPoll)
	api.POST("/polls/:slug/votes", h.vote)
	api.POST("/polls/:slug/close", h.requireUser(), h.closePoll)
	api.GET("/polls/:slug/events", h.events)
}

type signUpRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string            `json:"token"`
	User  domain.PublicUser `json:"user"`
}

func (h *Handler) signUp(c *gin.Context) {
	var request signUpRequest
	if !decodeJSON(c, &request) {
		return
	}
	user, token, err := h.auth.SignUp(c.Request.Context(), request.Name, request.Email, request.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, authResponse{Token: token, User: user.Public()})
}

func (h *Handler) login(c *gin.Context) {
	var request loginRequest
	if !decodeJSON(c, &request) {
		return
	}
	user, token, err := h.auth.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, authResponse{Token: token, User: user.Public()})
}

func (h *Handler) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": currentUser(c).Public()})
}

type createPollRequest struct {
	Question string     `json:"question"`
	Options  []string   `json:"options"`
	ClosesAt *time.Time `json:"closesAt"`
}

func (h *Handler) createPoll(c *gin.Context) {
	var request createPollRequest
	if !decodeJSON(c, &request) {
		return
	}
	poll, err := h.polls.Create(c.Request.Context(), currentUser(c).ID, request.Question, request.Options, request.ClosesAt)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"poll": poll})
}

func (h *Handler) listPolls(c *gin.Context) {
	polls, err := h.polls.ListForOwner(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"polls": polls})
}

func (h *Handler) getPoll(c *gin.Context) {
	viewerID := optionalUserID(c, h.auth)
	poll, err := h.polls.Get(c.Request.Context(), c.Param("slug"), viewerID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"poll": poll})
}

type voteRequest struct {
	OptionID string `json:"optionId"`
}

func (h *Handler) vote(c *gin.Context) {
	var request voteRequest
	if !decodeJSON(c, &request) {
		return
	}
	if _, err := uuid.Parse(request.OptionID); err != nil {
		writeError(c, http.StatusBadRequest, "select a valid option")
		return
	}
	voterID, err := c.Cookie("pulsepoll_voter")
	if err != nil || !validVoterID(voterID) {
		voterID = uuid.NewString()
		c.SetSameSite(h.cookieSameSite)
		c.SetCookie("pulsepoll_voter", voterID, 60*60*24*365, "/", "", h.cookieSecure, true)
	}
	poll, err := h.polls.Vote(c.Request.Context(), c.Param("slug"), request.OptionID, voterID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"poll": poll})
}

func (h *Handler) closePoll(c *gin.Context) {
	poll, err := h.polls.Close(c.Request.Context(), c.Param("slug"), currentUser(c).ID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"poll": poll})
}

func (h *Handler) events(c *gin.Context) {
	// First resolve the poll, then subscribe, then fetch the initial snapshot.
	// This order closes the otherwise possible vote gap during SSE connection.
	poll, err := h.polls.Get(c.Request.Context(), c.Param("slug"), optionalUserID(c, h.auth))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	subscription, err := h.broker.Subscribe(c.Request.Context(), poll.ID)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "live updates are temporarily unavailable")
		return
	}
	defer subscription.Close()

	// Re-fetch after subscribing, so the first event is always at least as new
	// as any event that could be published while the connection was opening.
	poll, err = h.polls.Get(c.Request.Context(), c.Param("slug"), optionalUserID(c, h.auth))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	if !sendEvent(c.Writer, "results", poll) {
		return
	}
	c.Writer.Flush()

	updates := subscription.Channel()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		case update, open := <-updates:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(c.Writer, "event: results\ndata: %s\n\n", update.Payload); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

func (h *Handler) requireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(c, http.StatusUnauthorized, "authentication is required")
			c.Abort()
			return
		}
		user, err := h.auth.UserFromToken(c.Request.Context(), parts[1])
		if err != nil {
			writeError(c, http.StatusUnauthorized, "your session is invalid or expired")
			c.Abort()
			return
		}
		c.Set(userContextKey, user)
		c.Next()
	}
}

func currentUser(c *gin.Context) domain.User {
	user, _ := c.MustGet(userContextKey).(domain.User)
	return user
}

func optionalUserID(c *gin.Context, auth *service.AuthService) *primitive.ObjectID {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil
	}
	user, err := auth.UserFromToken(c.Request.Context(), parts[1])
	if err != nil {
		return nil
	}
	return &user.ID
}

func decodeJSON(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	if err := c.ShouldBindJSON(target); err != nil {
		writeError(c, http.StatusBadRequest, "check the submitted fields and try again")
		return false
	}
	return true
}

func sendEvent(writer http.ResponseWriter, name string, payload any) bool {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	_, err = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", name, encoded)
	return err == nil
}

func validVoterID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, service.ErrEmailTaken):
		writeError(c, http.StatusConflict, "an account with this email already exists")
	case errors.Is(err, service.ErrPollNotFound), errors.Is(err, repository.ErrNotFound):
		writeError(c, http.StatusNotFound, "poll not found")
	case errors.Is(err, repository.ErrNotAuthorized):
		writeError(c, http.StatusForbidden, "you do not have access to manage this poll")
	case errors.Is(err, service.ErrPollClosed), errors.Is(err, service.ErrAlreadyVoted):
		writeError(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidOption):
		writeError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrRealtimeUnavailable):
		writeError(c, http.StatusServiceUnavailable, err.Error())
	default:
		// Known validation errors use safe, user-oriented text. Repository and
		// driver failures deliberately become a generic response.
		message := err.Error()
		if isValidationMessage(message) {
			writeError(c, http.StatusBadRequest, message)
			return
		}
		writeError(c, http.StatusInternalServerError, "something went wrong; please try again")
	}
}

func isValidationMessage(message string) bool {
	prefixes := []string{"name must", "enter a valid", "password must", "question must", "a poll needs", "each option", "options must", "closing time"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// CORS is intentionally narrow. Wildcard origins and credentialed cookies are
// never combined, which protects the anonymous vote cookie.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
