package services

import (
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityService struct {
	db *gorm.DB
	followService *FollowService
	collectionService *CollectionService // Add CollectionService
}

func NewCommunityService(db *gorm.DB) *CommunityService {
	return &CommunityService{
		db: db,
		followService: NewFollowService(db),
		collectionService: NewCollectionService(db), // Initialize CollectionService
	}
}

func (s *CommunityService) CreatePost(userID uuid.UUID, content, tag, imageURL string) (*models.Post, error) {
	post := models.Post{
		UserID:        userID,
		Content:       content,
		Tag:           tag,
		Image:         imageURL, // 保存图片URL
		LikesCount:    0,
		CommentsCount: 0,
	}

	if err := s.db.Create(&post).Error; err != nil {
		return nil, err
	}

	return &post, nil
}

func (s *CommunityService) GetPosts(page, pageSize int, sortBy, tag string, userID *uuid.UUID) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := s.db.Model(&models.Post{})

	// Filtering by tag
	if tag != "" {
		query = query.Where("tag = ?", tag)
	}

	// Count total before applying pagination
	query.Count(&total)

	// Sorting logic
	switch sortBy {
	case "likes_count":
		query = query.Order("likes_count DESC")
	case "comments_count":
		query = query.Order("comments_count DESC")
	default: // Default to created_at DESC
		query = query.Order("created_at DESC")
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Likes").Offset(offset).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	// 如果提供了用户ID，标记用户是否已点赞和是否关注了作者
	if userID != nil {
		for i := range posts {
			for _, like := range posts[i].Likes {
				if like.UserID == *userID {
					posts[i].IsLiked = true
					break
				}
			}
			// 检查当前用户是否关注了帖子作者
			if posts[i].User.ID != *userID { // 排除自己关注自己的情况
				isFollowing, err := s.followService.IsFollowing(*userID, posts[i].User.ID)
				if err != nil {
					// 记录错误，但不中断流程
					// log.Printf("Error checking follow status for user %s: %v", posts[i].User.ID, err)
				}
				posts[i].User.IsFollowing = isFollowing
			}
			// 检查当前用户是否收藏了帖子
			isCollected, err := s.collectionService.IsCollected(*userID, posts[i].ID)
			if err != nil {
				// 记录错误，但不中断流程
				// log.Printf("Error checking collection status for post %s: %v", posts[i].ID, err)
			}
			posts[i].IsCollected = isCollected
		}
	}

	return posts, total, nil
}

func (s *CommunityService) GetPost(postID uuid.UUID) (*models.Post, error) {
	var post models.Post
	if err := s.db.Preload("User").Preload("Likes").Preload("Comments.User").First(&post, postID).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *CommunityService) ToggleLike(userID, postID uuid.UUID) (bool, error) {
	var like models.PostLike
	err := s.db.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		// 点赞
		like = models.PostLike{
			UserID: userID,
			PostID: postID,
		}
		s.db.Create(&like)
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))
		return true, nil
	} else if err == nil {
		// 取消点赞
		s.db.Delete(&like)
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1"))
		return false, nil
	}

	return false, err
}

func (s *CommunityService) CreateComment(userID, postID uuid.UUID, content string) (*models.Comment, error) {
	comment := models.Comment{
		UserID:  userID,
		PostID:  postID,
		Content: content,
	}

	if err := s.db.Create(&comment).Error; err != nil {
		return nil, err
	}

	s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1"))

	return &comment, nil
}

func (s *CommunityService) GetComments(postID uuid.UUID) ([]models.Comment, error) {
	var comments []models.Comment
	if err := s.db.Preload("User").Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (s *CommunityService) DeletePost(userID, postID uuid.UUID) error {
	var post models.Post
	if err := s.db.First(&post, postID).Error; err != nil {
		return err // gorm.ErrRecordNotFound if not found
	}

	if post.UserID != userID {
		return gorm.ErrRecordNotFound // Or a custom error for unauthorized
	}

	// Delete associated comments and their likes
	var comments []models.Comment
	if err := s.db.Where("post_id = ?", postID).Find(&comments).Error; err != nil {
		return err
	}
	for _, comment := range comments {
		if err := s.db.Where("comment_id = ?", comment.ID).Delete(&models.CommentLike{}).Error; err != nil {
			return err
		}
	}
	if err := s.db.Where("post_id = ?", postID).Delete(&models.Comment{}).Error; err != nil {
		return err
	}

	// Delete associated post likes
	if err := s.db.Where("post_id = ?", postID).Delete(&models.PostLike{}).Error; err != nil {
		return err
	}

	// Delete associated post collections
	if err := s.db.Where("post_id = ?", postID).Delete(&models.PostCollection{}).Error; err != nil {
		return err
	}

	// Delete the post
	if err := s.db.Delete(&post).Error; err != nil {
		return err
	}

	return nil
}

func (s *CommunityService) DeleteComment(userID, commentID uuid.UUID) error {
	var comment models.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		return err // gorm.ErrRecordNotFound if not found
	}

	if comment.UserID != userID {
		return gorm.ErrRecordNotFound // Or a custom error for unauthorized
	}

	// Delete associated comment likes
	if err := s.db.Where("comment_id = ?", commentID).Delete(&models.CommentLike{}).Error; err != nil {
		return err
	}

	// Delete the comment
	if err := s.db.Delete(&comment).Error; err != nil {
		return err
	}

	// Decrement comments_count in the associated post
	s.db.Model(&models.Post{}).Where("id = ?", comment.PostID).UpdateColumn("comments_count", gorm.Expr("comments_count - 1"))

	return nil
}

func (s *CommunityService) ToggleCommentLike(userID, commentID uuid.UUID) (bool, error) {
	var like models.CommentLike
	err := s.db.Where("user_id = ? AND comment_id = ?", userID, commentID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		// 点赞
		like = models.CommentLike{
			UserID:    userID,
			CommentID: commentID,
		}
		s.db.Create(&like)
		s.db.Model(&models.Comment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))
		return true, nil
	} else if err == nil {
		// 取消点赞
		s.db.Delete(&like)
		s.db.Model(&models.Comment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1"))
		return false, nil
	}

	return false, err
}

func (s *CommunityService) GetUserPosts(targetUserID uuid.UUID, page, pageSize int, currentUserID *uuid.UUID) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	query := s.db.Model(&models.Post{}).Where("user_id = ?", targetUserID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Likes").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	// 如果提供了当前用户ID，标记用户是否已点赞和是否关注了作者
	if currentUserID != nil {
		for i := range posts {
			for _, like := range posts[i].Likes {
				if like.UserID == *currentUserID {
					posts[i].IsLiked = true
					break
				}
			}
			// 检查当前用户是否关注了帖子作者
			if posts[i].User.ID != *currentUserID { // 排除自己关注自己的情况
				isFollowing, err := s.followService.IsFollowing(*currentUserID, posts[i].User.ID)
				if err != nil {
					// 记录错误，但不中断流程
					// log.Printf("Error checking follow status for user %s: %v", posts[i].User.ID, err)
				}
				posts[i].User.IsFollowing = isFollowing
			}
			// 检查当前用户是否收藏了帖子
			isCollected, err := s.collectionService.IsCollected(*currentUserID, posts[i].ID)
			if err != nil {
				// 记录错误，但不中断流程
				// log.Printf("Error checking collection status for post %s: %v", posts[i].ID, err)
			}
			posts[i].IsCollected = isCollected
		}
	}

	return posts, total, nil
}


