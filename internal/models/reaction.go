package models

import (
	"database/sql"
	"errors"
)

type Reaction struct {
	ReactionID int
	UserID     int
	PostID     int
	LikeStatus int
	Username   string
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type ReactionModel struct {
	DB *sql.DB
}

func (r *ReactionModel) GetLikes(id int) (int, error) {
	stmt := `SELECT COUNT(like_status) FROM reactions WHERE like_status = 1 AND post_id = ?`

	row := r.DB.QueryRow(stmt, id)

	var likes int // Declare a variable to hold the result

	err := row.Scan(&likes) // Use &likes to scan the result into the variable
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNoRecord
		} else {
			return 0, err
		}
	}

	return likes, nil
}

func (r *ReactionModel) GetDislikes(id int) (int, error) {
	stmt := `SELECT COUNT(like_status) FROM reactions WHERE like_status = -1 AND post_id = ?`

	row := r.DB.QueryRow(stmt, id)

	var dislikes int // Declare a variable to hold the result

	err := row.Scan(&dislikes) // Use &likes to scan the result into the variable
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNoRecord
		} else {
			return 0, err
		}
	}

	return dislikes, nil
}

func (r *ReactionModel) MakeReaction(user_id, post_id, like_status int) error {
	// Check if the user has already reacted to the post
	stmtSelect := `SELECT like_status FROM reactions WHERE user_id = ? AND post_id = ?`
	var existingLikeStatus int
	err := r.DB.QueryRow(stmtSelect, user_id, post_id).Scan(&existingLikeStatus)

	if err != nil {
		// No existing reaction, insert a new one
		if err == sql.ErrNoRows {
			stmtInsert := `INSERT INTO reactions (user_id, post_id, like_status) VALUES (?, ?, ?)`
			_, err := r.DB.Exec(stmtInsert, user_id, post_id, like_status)
			if err != nil {
				return err
			}
		} else {
			// An error occurred during the SELECT query
			return err
		}
	} else {
		// User has already reacted, toggle the like/dislike status
		if existingLikeStatus == like_status {
			// If the existing like_status is the same as the new one, remove the reaction
			stmtDelete := `DELETE FROM reactions WHERE user_id = ? AND post_id = ?`
			_, err := r.DB.Exec(stmtDelete, user_id, post_id)
			if err != nil {
				return err
			}
		} else {
			// If the existing like_status is different, update the existing reaction
			stmtUpdate := `UPDATE reactions SET like_status = ? WHERE user_id = ? AND post_id = ?`
			_, err := r.DB.Exec(stmtUpdate, like_status, user_id, post_id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ReactionModel) GetPostReactions(user_id int) ([]*Reaction, error) {
	stmt := `SELECT reactions.user_id, reactions.post_id, reactions.like_status, users.username
    FROM reactions
    JOIN users ON reactions.user_id = users.user_id
    WHERE reactions.post_id IN (
        SELECT post_id
        FROM posts
        WHERE user_id = ?
    ) AND reactions.user_id != ?`

	rows, err := r.DB.Query(stmt, user_id, user_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reactions := []*Reaction{}
	for rows.Next() {
		r := &Reaction{}
		err = rows.Scan(&r.UserID, &r.PostID, &r.LikeStatus, &r.Username)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reactions, nil
}
