package models

import (
	"database/sql"
)

type PostCategory struct {
	PostID     int
	CategoryID int
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type CategoryModel struct {
	DB *sql.DB
}

// This will insert a new User into the database.
func (c *CategoryModel) InsertCategory(PostID int, CategoryName string) error {
	CategoryID, err := c.getTagIDByName(CategoryName)
	stmt := `INSERT INTO posts_categories (post_id, category_id)
	VALUES (?, ?)`
	_, err = c.DB.Exec(stmt, PostID, CategoryID)
	if err != nil {
		return err
	}
	return nil
}

func (c *CategoryModel) getTagIDByName(categoryName string) (int, error) {
	var categoryID int
	err := c.DB.QueryRow("SELECT category_id FROM categories WHERE category = ?", categoryName).Scan(&categoryID)
	if err != nil {
		return 0, err
	}
	return categoryID, nil
}

func (c *CategoryModel) GetCategoriesByPostID(postID int) ([]string, error) {
	stmt := `SELECT c.category FROM categories c
             INNER JOIN posts_categories pc ON c.category_id = pc.category_id
             WHERE pc.post_id = ?`

	rows, err := c.DB.Query(stmt, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []string{}

	for rows.Next() {
		var category string
		err = rows.Scan(&category)
		if err != nil {
			return nil, err
		}

		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}
