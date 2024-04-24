package models

import (
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"strings"
	"time"
)

type Post struct {
	ID          int
	Title       string
	Content     string
	Created     time.Time
	UserId      int
	Author      string
	Likes       int
	Dislikes    int
	Categories  []string
	Image       string
	Currentuser int
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type PostModel struct {
	DB *sql.DB
}

// This will insert a new User into the database.
func (p *PostModel) Insert(title string, content string, userId int) (int, error) {
	stmt := `INSERT INTO posts (title, content, created_date, user_id)
	VALUES (?, ?, datetime('now','localtime'), ?)`
	result, err := p.DB.Exec(stmt, title, content, userId)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (p *PostModel) Get(id int) (*Post, error) {
	stmt := `SELECT p.post_id, p.title, p.content, p.created_date, u.user_id, u.username FROM posts p 
	INNER JOIN users u ON p.user_id = u.user_id
	WHERE post_id = ?`

	row := p.DB.QueryRow(stmt, id)

	post := &Post{}

	err := row.Scan(&post.ID, &post.Title, &post.Content, &post.Created, &post.UserId, &post.Author)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}
	// Применяем экранирование HTML
	post.Title = html.EscapeString(post.Title)
	post.Content = html.EscapeString(post.Content)
	post.Author = html.EscapeString(post.Author)

	return post, nil
}

func (p *PostModel) GetUserPosts(userID int) ([]*Post, error) {
	stmt := `SELECT posts.post_id, posts.title, posts.content, posts.created_date, posts.user_id,
    	COALESCE(SUM(reactions.like_status), 0) AS total_likes,
    	COALESCE(images.image_hash, 'NoImage') AS image_hash,
    	COALESCE(images.file_type, 'jpg') AS file_type
		FROM posts
		LEFT JOIN reactions ON posts.post_id = reactions.post_id
		LEFT JOIN images ON posts.post_id = images.post_id
		WHERE posts.user_id = ?
		GROUP BY posts.post_id, posts.title, posts.content, posts.created_date, posts.user_id, images.image_hash, images.file_type
		ORDER BY posts.created_date DESC`
	rows, err := p.DB.Query(stmt, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	posts := []*Post{}
	for rows.Next() {
		post := &Post{}
		image := &Image{}

		err = rows.Scan(&post.ID, &post.Title, &post.Content, &post.Created, &post.UserId, &post.Likes, &image.ImageHash, &image.FileType)
		if err != nil {
			return nil, err
		}
		post.Image = fmt.Sprintf("/static/img/%s.%s", image.ImageHash, image.FileType)
		// Применяем экранирование HTML
		post.Title = html.EscapeString(post.Title)
		post.Content = html.EscapeString(post.Content)

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (p *PostModel) GetUserLikedPosts(userID int) ([]*Post, error) {
	stmt := `SELECT posts.post_id, posts.title, posts.content, posts.created_date, posts.user_id,
    	COALESCE(SUM(reactions.like_status), 0) AS total_likes,
    	COALESCE(images.image_hash, 'NoImage') AS image_hash,
    	COALESCE(images.file_type, 'jpg') AS file_type
		FROM posts
		LEFT JOIN reactions ON posts.post_id = reactions.post_id
		LEFT JOIN images ON posts.post_id = images.post_id
		WHERE reactions.like_status = 1
        AND reactions.user_id = ?
		GROUP BY posts.post_id, posts.title, posts.content, posts.created_date, posts.user_id, images.image_hash, images.file_type
		ORDER BY posts.created_date DESC`
	rows, err := p.DB.Query(stmt, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	posts := []*Post{}
	for rows.Next() {
		post := &Post{}
		image := &Image{}

		err = rows.Scan(&post.ID, &post.Title, &post.Content, &post.Created, &post.UserId, &post.Likes, &image.ImageHash, &image.FileType)
		if err != nil {
			return nil, err
		}
		post.Image = fmt.Sprintf("/static/img/%s.%s", image.ImageHash, image.FileType)

		// Применяем экранирование HTML
		post.Title = html.EscapeString(post.Title)
		post.Content = html.EscapeString(post.Content)

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (p *PostModel) Latest() ([]*Post, error) {
	stmt := `SELECT posts.post_id, posts.title, posts.content, posts.created_date, posts.user_id, COALESCE(SUM(reactions.like_status), 0),     COALESCE(images.image_hash, 'NoImage') AS image_hash,
    COALESCE(images.file_type, 'jpg') AS file_type
	FROM posts
	LEFT JOIN reactions ON posts.post_id = reactions.post_id
	LEFT JOIN images ON posts.post_id = images.post_id
	GROUP BY posts.post_id
	ORDER BY posts.created_date DESC`
	rows, err := p.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := []*Post{}
	for rows.Next() {
		post := &Post{}
		image := &Image{}
		err = rows.Scan(&post.ID, &post.Title, &post.Content, &post.Created, &post.UserId, &post.Likes, &image.ImageHash, &image.FileType)
		if err != nil {
			return nil, err
		}
		post.Image = fmt.Sprintf("/static/img/%s.%s", image.ImageHash, image.FileType)

		// Применяем экранирование HTML
		post.Title = html.EscapeString(post.Title)
		post.Content = html.EscapeString(post.Content)

		posts = append(posts, post)

	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

// getTagIDByName принимает имя категории и возвращает соответствующий идентификатор категории (tag_id) из базы данных.
func (p *PostModel) LatestWithCategory(categories []string) ([]*Post, error) {
	// Create the query with dynamic placeholders
	placeholders := make([]string, len(categories))
	for i := range categories {
		placeholders[i] = "?"
	}
	categoryPlaceholders := strings.Join(placeholders, ",")

	stmt := `
SELECT
    posts.post_id,
    posts.title,
    posts.content,
    posts.created_date,
    posts.user_id,
    COALESCE(SUM(reactions.like_status), 0) AS like_count,
    COALESCE(images.image_hash, 'NoImage') AS image_hash,
    COALESCE(images.file_type, 'jpg') AS file_type
FROM
    posts
LEFT JOIN reactions ON posts.post_id = reactions.post_id
LEFT JOIN images ON posts.post_id = images.post_id
WHERE
    posts.post_id IN (
        SELECT
            post_id
        FROM
            posts_categories
        WHERE
            category_id IN (
                SELECT
                    category_id
                FROM
                    categories
                WHERE
                    category IN (` + categoryPlaceholders + `)
            )
    )
GROUP BY
    posts.post_id
ORDER BY
    posts.post_id DESC;`

	// Convert the []string slice to []interface{} slice
	args := make([]interface{}, len(categories))
	for i, v := range categories {
		args[i] = v
	}

	rows, err := p.DB.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []*Post{}
	for rows.Next() {
		post := &Post{}
		image := &Image{} // Assuming Image struct is defined in your code

		err = rows.Scan(&post.ID, &post.Title, &post.Content, &post.Created, &post.UserId, &post.Likes, &image.ImageHash, &image.FileType)
		if err != nil {
			return nil, err
		}
		post.Image = fmt.Sprintf("/static/img/%s.%s", image.ImageHash, image.FileType)

		// Применяем экранирование HTML
		post.Title = html.EscapeString(post.Title)
		post.Content = html.EscapeString(post.Content)

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (p *PostModel) DeletePost(postID, userID int) error {
	stmt := `DELETE FROM posts WHERE post_id = ? AND user_id = ?;
	DELETE FROM comments WHERE post_id = ? and user_id = ?;
	`
	_, err := p.DB.Exec(stmt, postID, userID, postID, userID)
	return err
}

func (p *PostModel) UpdatePost(title, content string, postID, userID int) error {
	// Проверка, что пользователь имеет права на изменение этого поста
	var dbUserID int
	err := p.DB.QueryRow("SELECT user_id FROM posts WHERE post_id = ?", postID).Scan(&dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("пост с ID %d не найден", postID)
		}
		return err
	}

	if dbUserID != userID {
		return fmt.Errorf("это не ваш пост, вы не можете изменить его")
	}

	stmt := `UPDATE posts SET title = ?, content = ? WHERE post_id = ? AND user_id = ?;`
	_, err = p.DB.Exec(stmt, title, content, postID, userID)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (p *PostModel) GetUserID(id int) (int, error) {
	stmt := `SELECT user_id FROM posts WHERE post_id = ?`

	row := p.DB.QueryRow(stmt, id)

	var userID int

	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNoRecord
		} else {
			return 0, err
		}
	}
	return userID, nil
}
