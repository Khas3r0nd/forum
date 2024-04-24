package models

import (
	"database/sql"
	"errors"
)

type ReactionComment struct {
	ReactionID int
	UserID     int
	PostID     int
	LikeStatus int
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type ReactionCommentModel struct {
	DB *sql.DB
}

func (r *ReactionModel) GetCommentLikes(id int) (int, error) {
	stmt := `SELECT COUNT(like_status) FROM comment_reactions WHERE like_status = 1 AND comment_id = ?`

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

func (r *ReactionModel) GetCommentDislikes(id int) (int, error) {
	stmt := `SELECT COUNT(like_status) FROM comment_reactions WHERE like_status = -1 AND comment_id = ?`

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

func (r *ReactionModel) MakeCommentReaction(user_id, comment_id, like_status int) error {
	// Check if the user has already reacted to the post
	stmtSelect := `SELECT like_status FROM comment_reactions WHERE user_id = ? AND comment_id = ?`
	var existingLikeStatus int
	err := r.DB.QueryRow(stmtSelect, user_id, comment_id).Scan(&existingLikeStatus)

	if err != nil {
		// No existing reaction, insert a new one
		if err == sql.ErrNoRows {
			stmtInsert := `INSERT INTO comment_reactions (user_id, comment_id, like_status) VALUES (?, ?, ?)`
			_, err := r.DB.Exec(stmtInsert, user_id, comment_id, like_status)
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
			stmtDelete := `DELETE FROM comment_reactions WHERE user_id = ? AND comment_id = ?`
			_, err := r.DB.Exec(stmtDelete, user_id, comment_id)
			if err != nil {
				return err
			}
		} else {
			// If the existing like_status is different, update the existing reaction
			stmtUpdate := `UPDATE comment_reactions SET like_status = ? WHERE user_id = ? AND comment_id = ?`
			_, err := r.DB.Exec(stmtUpdate, like_status, user_id, comment_id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
