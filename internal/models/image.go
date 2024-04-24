package models

import (
	"database/sql"
	"errors"
	"fmt"
)

type Image struct {
	ImageID   int
	PostID    int
	ImageHash string
	FileType  string
}

// Define a SnippetModel type which wraps a sql.DB connection pool.
type ImageModel struct {
	DB *sql.DB
}

func (i *ImageModel) InsertImage(post_id int, image_hash, file_type string) error {
	stmt := `INSERT INTO images (post_id, image_hash, file_type)
	VALUES (?, ?, ?)`
	_, err := i.DB.Exec(stmt, post_id, image_hash, file_type)
	if err != nil {
		fmt.Println(err, "TUT")
		return err
	}
	return nil
}

func (i *ImageModel) GetImage(id int) (string, string, error) {
	stmt := `SELECT image_hash, file_type FROM images
    LEFT JOIN posts ON posts.post_id = images.post_id
    WHERE images.post_id = ?`

	row := i.DB.QueryRow(stmt, id, id)

	image := &Image{}

	err := row.Scan(&image.ImageHash, &image.FileType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrNoRecord
		} else {
			return "", "", err
		}
	}
	return image.ImageHash, image.FileType, nil
}
