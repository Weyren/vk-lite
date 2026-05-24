package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Weyren/vk-lite/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SocialHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	events EventPublisher
}

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload any)
}

type postResponse struct {
	ID         int64     `json:"id"`
	AuthorID   int64     `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Content    string    `json:"content"`
	MediaURL   string    `json:"media_url"`
	LikesCount int64     `json:"likes_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewSocialHandler(db *gorm.DB, redisClient *redis.Client, events EventPublisher) *SocialHandler {
	return &SocialHandler{db: db, redis: redisClient, events: events}
}

func (h *SocialHandler) GetUser(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var user models.User
	if err := h.db.WithContext(c.Request.Context()).First(&user, id).Error; err != nil {
		statusFromDBError(c, err)
		return
	}

	var followers, following, posts int64
	ctx := c.Request.Context()
	h.db.WithContext(ctx).Model(&models.Subscription{}).Where("target_id = ?", id).Count(&followers)
	h.db.WithContext(ctx).Model(&models.Subscription{}).Where("subscriber_id = ?", id).Count(&following)
	h.db.WithContext(ctx).Model(&models.Post{}).Where("author_id = ?", id).Count(&posts)

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"user": gin.H{
			"id":              user.ID,
			"email":           user.Email,
			"name":            user.Name,
			"avatar_url":      user.AvatarURL,
			"followers_count": followers,
			"following_count": following,
			"posts_count":     posts,
			"created_at":      user.CreatedAt,
			"updated_at":      user.UpdatedAt,
		},
	})
}

func (h *SocialHandler) ToggleFollow(c *gin.Context) {
	currentUserID := currentUserID(c)
	targetID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if currentUserID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you cannot follow yourself"})
		return
	}

	ctx := c.Request.Context()
	if err := h.db.WithContext(ctx).First(&models.User{}, targetID).Error; err != nil {
		statusFromDBError(c, err)
		return
	}

	subscription := models.Subscription{SubscriberID: currentUserID, TargetID: targetID}
	err := h.db.WithContext(ctx).First(&subscription, "subscriber_id = ? AND target_id = ?", currentUserID, targetID).Error
	if err == nil {
		if err := h.db.WithContext(ctx).Delete(&subscription).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		h.invalidateFeed(ctx, currentUserID)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "following": false})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.WithContext(ctx).Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.invalidateFeed(ctx, currentUserID)
	h.publish(ctx, "user.followed", gin.H{"subscriber_id": currentUserID, "target_id": targetID})
	c.JSON(http.StatusOK, gin.H{"status": "ok", "following": true})
}

func (h *SocialHandler) CreatePost(c *gin.Context) {
	var req struct {
		Content  string `json:"content"`
		MediaURL string `json:"media_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Content == "" && req.MediaURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content or media_url is required"})
		return
	}

	post := models.Post{
		AuthorID: currentUserID(c),
		Content:  req.Content,
		MediaURL: req.MediaURL,
	}
	if err := h.db.WithContext(c.Request.Context()).Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.pushPostToFeeds(c.Request.Context(), post)
	h.publish(c.Request.Context(), "post.created", gin.H{"post_id": post.ID, "author_id": post.AuthorID})

	response, err := h.postResponse(c.Request.Context(), post)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "ok", "post": response})
}

func (h *SocialHandler) ToggleLike(c *gin.Context) {
	postID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	if err := h.db.WithContext(ctx).First(&models.Post{}, postID).Error; err != nil {
		statusFromDBError(c, err)
		return
	}

	userID := currentUserID(c)
	like := models.Like{UserID: userID, PostID: postID}
	err := h.db.WithContext(ctx).First(&like, "user_id = ? AND post_id = ?", userID, postID).Error
	liked := false
	if err == nil {
		if err := h.db.WithContext(ctx).Delete(&like).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := h.db.WithContext(ctx).Create(&like).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		liked = true
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	likesCount := h.countLikesFromDB(ctx, postID)
	h.cacheLikes(ctx, postID, likesCount)
	if liked {
		h.publish(ctx, "post.liked", gin.H{"post_id": postID, "user_id": userID})
	} else {
		h.publish(ctx, "post.unliked", gin.H{"post_id": postID, "user_id": userID})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "liked": liked, "likes_count": likesCount})
}

func (h *SocialHandler) GetFeed(c *gin.Context) {
	page := positiveQueryInt(c, "page", 1)
	perPage := positiveQueryInt(c, "per_page", 20)
	if perPage > 50 {
		perPage = 50
	}

	ctx := c.Request.Context()
	userID := currentUserID(c)
	offset := (page - 1) * perPage

	posts, fromCache, err := h.feedFromCache(ctx, userID, offset, perPage)
	if err != nil || len(posts) == 0 {
		posts, err = h.feedFromDB(ctx, userID, offset, perPage)
		fromCache = false
		if err == nil {
			h.warmFeedCache(ctx, userID)
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"page":       page,
		"per_page":   perPage,
		"from_cache": fromCache,
		"posts":      posts,
	})
}

func (h *SocialHandler) feedFromDB(ctx context.Context, userID int64, offset, limit int) ([]postResponse, error) {
	var rows []postResponse
	err := h.db.WithContext(ctx).
		Table("posts").
		Select("posts.id, posts.author_id, users.name AS author_name, posts.content, posts.media_url, posts.created_at, posts.updated_at").
		Joins("JOIN users ON users.id = posts.author_id").
		Where("posts.author_id = ? OR posts.author_id IN (?)",
			userID,
			h.db.Table("subscriptions").Select("target_id").Where("subscriber_id = ?", userID),
		).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	h.fillLikeCounts(ctx, rows)
	return rows, nil
}

func (h *SocialHandler) feedFromCache(ctx context.Context, userID int64, offset, limit int) ([]postResponse, bool, error) {
	if h.redis == nil {
		return nil, false, nil
	}

	ids, err := h.redis.ZRevRange(ctx, feedKey(userID), int64(offset), int64(offset+limit-1)).Result()
	if err != nil || len(ids) == 0 {
		return nil, false, err
	}

	idNums := make([]int64, 0, len(ids))
	for _, id := range ids {
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err == nil {
			idNums = append(idNums, parsed)
		}
	}
	if len(idNums) == 0 {
		return nil, false, nil
	}

	var rows []postResponse
	if err := h.db.WithContext(ctx).
		Table("posts").
		Select("posts.id, posts.author_id, users.name AS author_name, posts.content, posts.media_url, posts.created_at, posts.updated_at").
		Joins("JOIN users ON users.id = posts.author_id").
		Where("posts.id IN ?", idNums).
		Scan(&rows).Error; err != nil {
		return nil, false, err
	}

	byID := make(map[int64]postResponse, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	ordered := make([]postResponse, 0, len(rows))
	for _, id := range idNums {
		if row, ok := byID[id]; ok {
			ordered = append(ordered, row)
		}
	}
	h.fillLikeCounts(ctx, ordered)
	return ordered, true, nil
}

func (h *SocialHandler) warmFeedCache(ctx context.Context, userID int64) {
	if h.redis == nil {
		return
	}

	var posts []models.Post
	err := h.db.WithContext(ctx).
		Where("author_id = ? OR author_id IN (?)",
			userID,
			h.db.Table("subscriptions").Select("target_id").Where("subscriber_id = ?", userID),
		).
		Order("created_at DESC").
		Limit(100).
		Find(&posts).Error
	if err != nil {
		return
	}

	key := feedKey(userID)
	h.redis.Del(ctx, key)
	for _, post := range posts {
		h.redis.ZAdd(ctx, key, redis.Z{Score: float64(post.CreatedAt.Unix()), Member: strconv.FormatInt(post.ID, 10)})
	}
	h.redis.Expire(ctx, key, 10*time.Minute)
}

func (h *SocialHandler) pushPostToFeeds(ctx context.Context, post models.Post) {
	if h.redis == nil {
		return
	}

	h.redis.ZAdd(ctx, feedKey(post.AuthorID), redis.Z{Score: float64(post.CreatedAt.Unix()), Member: strconv.FormatInt(post.ID, 10)})
	h.redis.Expire(ctx, feedKey(post.AuthorID), 10*time.Minute)

	var subscriberIDs []int64
	if err := h.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("target_id = ?", post.AuthorID).
		Pluck("subscriber_id", &subscriberIDs).Error; err != nil {
		return
	}
	for _, subscriberID := range subscriberIDs {
		h.redis.ZAdd(ctx, feedKey(subscriberID), redis.Z{Score: float64(post.CreatedAt.Unix()), Member: strconv.FormatInt(post.ID, 10)})
		h.redis.Expire(ctx, feedKey(subscriberID), 10*time.Minute)
	}
}

func (h *SocialHandler) postResponse(ctx context.Context, post models.Post) (postResponse, error) {
	var author models.User
	if err := h.db.WithContext(ctx).First(&author, post.AuthorID).Error; err != nil {
		return postResponse{}, err
	}

	return postResponse{
		ID:         post.ID,
		AuthorID:   post.AuthorID,
		AuthorName: author.Name,
		Content:    post.Content,
		MediaURL:   post.MediaURL,
		LikesCount: h.countLikes(ctx, post.ID),
		CreatedAt:  post.CreatedAt,
		UpdatedAt:  post.UpdatedAt,
	}, nil
}

func (h *SocialHandler) fillLikeCounts(ctx context.Context, posts []postResponse) {
	for i := range posts {
		posts[i].LikesCount = h.countLikes(ctx, posts[i].ID)
	}
}

func (h *SocialHandler) countLikes(ctx context.Context, postID int64) int64 {
	if h.redis != nil {
		cached, err := h.redis.Get(ctx, likesKey(postID)).Int64()
		if err == nil {
			return cached
		}
	}

	return h.countLikesFromDB(ctx, postID)
}

func (h *SocialHandler) countLikesFromDB(ctx context.Context, postID int64) int64 {
	var count int64
	h.db.WithContext(ctx).Model(&models.Like{}).Where("post_id = ?", postID).Count(&count)
	return count
}

func (h *SocialHandler) cacheLikes(ctx context.Context, postID, count int64) {
	if h.redis != nil {
		h.redis.Set(ctx, likesKey(postID), count, 30*time.Minute)
	}
}

func (h *SocialHandler) invalidateFeed(ctx context.Context, userID int64) {
	if h.redis != nil {
		h.redis.Del(ctx, feedKey(userID))
	}
}

func (h *SocialHandler) publish(ctx context.Context, eventType string, payload any) {
	if h.events != nil {
		h.events.Publish(ctx, eventType, payload)
	}
}

func currentUserID(c *gin.Context) int64 {
	value, _ := c.Get("user_id")
	userID, _ := value.(int64)
	return userID
}

func parseIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}
	return id, true
}

func positiveQueryInt(c *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(c.DefaultQuery(name, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func statusFromDBError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func feedKey(userID int64) string {
	return "feed:" + strconv.FormatInt(userID, 10)
}

func likesKey(postID int64) string {
	return "post:likes:" + strconv.FormatInt(postID, 10)
}
