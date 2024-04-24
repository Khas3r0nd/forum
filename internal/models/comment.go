package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Comment struct {
	CommentID       int
	UserID          int
	Username        string
	PostID          int
	Text            string
	CreatedAt       time.Time
	CommentLikes    int
	CommentDislikes int
	CurrentUser     int
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type CommentModel struct {
	DB *sql.DB
}

func (m *CommentModel) Insert(user_id, post_id int, comment string) (int, error) {
	stmt := `INSERT INTO comments (user_id, post_id, comment, created_at)
	VALUES (?, ?, ?, datetime('now','localtime'))`
	result, err := m.DB.Exec(stmt, user_id, post_id, comment)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	// Use the LastInsertId() method on the result to get the ID of our
	// newly inserted record in the snippets table.
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	// The ID returned has the type int64, so we convert it to an int type
	// before returning.
	return int(id), nil
}

func (m *CommentModel) GetComments(post_id int) ([]*Comment, error) {
	stmt := `SELECT c.comment_id,  c.user_id, c.post_id, u.username, c.comment, c.created_at
				FROM comments c
				JOIN users u ON c.user_id = u.user_id 
				WHERE c.post_id = $1`
	rows, err := m.DB.Query(stmt, post_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []*Comment{}

	for rows.Next() {
		c := &Comment{}
		err = rows.Scan(&c.CommentID, &c.UserID, &c.PostID, &c.Username, &c.Text, &c.CreatedAt)
		if err != nil {
			return nil, err
		}

		comments = append(comments, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *CommentModel) GetComment(post_id int) (*Comment, error) {
	stmt := `SELECT c.comment_id, c.user_id, u.username, c.comment, c.created_at
             FROM comments c
             JOIN users u ON c.user_id = u.user_id 
             WHERE c.post_id = $1`
	rows, err := m.DB.Query(stmt, post_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := &Comment{}

	for rows.Next() {
		c := &Comment{}
		err = rows.Scan(&c.CommentID, &c.UserID, &c.Username, &c.Text, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *CommentModel) GetUserComments(user_id int) ([]*Comment, error) {
	stmt := `SELECT comment_id, user_id, post_id, comment, created_at
	FROM comments
	WHERE user_id = ?`
	rows, err := m.DB.Query(stmt, user_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []*Comment{}

	for rows.Next() {
		c := &Comment{}
		err = rows.Scan(&c.CommentID, &c.UserID, &c.PostID, &c.Text, &c.CreatedAt)
		if err != nil {
			return nil, err
		}

		comments = append(comments, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *CommentModel) CommentNotifications(user_id int) ([]*Comment, error) {
	stmt := `SELECT users.user_id, users.username, comments.post_id, comments.comment
	FROM comments
	JOIN posts ON comments.post_id = posts.post_id
	JOIN users ON comments.user_id = users.user_id
	WHERE posts.user_id = ?`
	rows, err := m.DB.Query(stmt, user_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []*Comment{}

	for rows.Next() {
		c := &Comment{}
		err = rows.Scan(&c.UserID, &c.Username, &c.PostID, &c.Text)
		if err != nil {
			return nil, err
		}

		comments = append(comments, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (m *CommentModel) GetCommentID(comment_id int) (*Comment, error) {
	stmt := `SELECT comment 
             FROM comments 
             WHERE comment_id = $1`
	rows, err := m.DB.Query(stmt, comment_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comment := &Comment{} // Изменил название переменной на singular, так как мы получаем только один комментарий по ID

	for rows.Next() {
		err = rows.Scan(&comment.Text)
		if err != nil {
			// Добавьте логирование ошибки, чтобы увидеть, что происходит
			log.Println("Error scanning comment:", err)
			return nil, err
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comment, nil
}

func (c *CommentModel) DeleteComment(commentID, userID int) error {
	stmt := `DELETE FROM comments WHERE comment_id = ? and user_id = ?`
	_, err := c.DB.Exec(stmt, commentID, userID)
	return err
}

func (c *CommentModel) EditComment(comment string, commentID, userID int) error {
	stmt := `UPDATE comments SET comment = ? WHERE comment_id = ? AND user_id = ?;`
	_, err := c.DB.Exec(stmt, comment, commentID, userID)
	return err
}

func (m *CommentModel) GetPostIDByCommentID(commentID int) (int, error) {
	stmt := `SELECT post_id FROM comments WHERE comment_id = $1`
	var postID int
	err := m.DB.QueryRow(stmt, commentID).Scan(&postID)
	if err != nil {
		return 0, err
	}
	return postID, nil
}

func (m *CommentModel) GetCommentUserID(comment_id int) (int, error) {
	stmt := `SELECT user_id 
             FROM comments 
             WHERE comment_id = $1`
	rows, err := m.DB.Query(stmt, comment_id)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	comment := &Comment{} // Изменил название переменной на singular, так как мы получаем только один комментарий по ID

	for rows.Next() {
		err = rows.Scan(&comment.UserID)
		if err != nil {
			// Добавьте логирование ошибки, чтобы увидеть, что происходит
			log.Println("Error scanning comment:", err)
			return 0, err
		}
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	return comment.UserID, nil
}
