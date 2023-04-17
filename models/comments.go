package models

import (
	"time"

	"github.com/upper/db/v4"
)

type Comment struct {
	ID        int       `db:"comment_id,omitempty"`
	CreatedAt time.Time `db:"created_at,omitempty"`
	Body      string    `db:"body"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	User      `db:",inline"`
}

type CommentsModel struct {
	db db.Session
}

func (m CommentsModel) GetForPost(postId int) ([]Comment, error) {
	var comments []Comment

	q := m.db.SQL().Select("c.id as comment_id", "c.created_at as c_created_at", "*").
	From("comments as C").
	Join("users as u").On("c.user_id = u.id").
	Where(db.Cond{"c.post_id": postId}).
	OrderBy("c.created desc")

	err := q.All(&comments)
	if err != nil {
		return nil, err
	}

	return comments, nil
}
